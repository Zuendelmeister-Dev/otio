"""Run with python -m unittest discover -s scripts -p historian_test.py."""
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
import uuid
from unittest import mock

import historian


class HistorianBackupTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.archive = self.directory / 'saved.dump'

    def test_backup_preserves_binary_and_publishes_only_complete_archive(self):
        payload = b'PGDMP\x00\xff\r\n\x1a\x80binary'
        database = mock.Mock()
        database.run.side_effect = lambda *args, **kwargs: kwargs['stdout'].write(payload)
        historian.backup(database, self.archive)
        self.assertEqual(self.archive.read_bytes(), payload)
        self.assertEqual(list(self.directory.iterdir()), [self.archive])
        flags = database.run.call_args.args[1]
        self.assertIn('--format=custom', flags)
        self.assertIn('--no-password', flags)

    def test_failure_removes_partial_output_and_existing_backups_are_preserved(self):
        database = mock.Mock()
        def failed_dump(*args, **kwargs):
            kwargs['stdout'].write(b'PGDMPpartial')
            raise historian.HistorianError('connection failed')
        database.run.side_effect = failed_dump
        with self.assertRaises(historian.HistorianError):
            historian.backup(database, self.archive)
        self.assertEqual(list(self.directory.iterdir()), [])
        self.archive.write_bytes(b'original')
        with self.assertRaisesRegex(historian.HistorianError, 'never overwritten'):
            historian.backup(database, self.archive)
        self.assertEqual(self.archive.read_bytes(), b'original')

    def test_invalid_or_competing_backup_never_publishes_or_clobbers(self):
        database = mock.Mock()
        database.run.side_effect = lambda *args, **kwargs: kwargs['stdout'].write(b'not an archive')
        with self.assertRaisesRegex(historian.HistorianError, 'PGDMP'):
            historian.backup(database, self.archive)
        self.assertFalse(self.archive.exists())
        def competing(*args, **kwargs):
            kwargs['stdout'].write(b'PGDMPcompleted')
            self.archive.write_bytes(b'other writer')
        database.run.side_effect = competing
        with self.assertRaisesRegex(historian.HistorianError, 'not overwritten'):
            historian.backup(database, self.archive)
        self.assertEqual(self.archive.read_bytes(), b'other writer')

    def test_restore_checks_target_and_uses_one_transaction_without_clean(self):
        self.archive.write_bytes(b'PGDMPbinary\x00\xff')
        database = mock.Mock()
        response = subprocess.CompletedProcess([], 0, json.dumps({'database': 'recovered', 'objects': 0}).encode(), b'')
        consumed = []
        def run(tool, flags, **kwargs):
            if kwargs.get('stdin'):
                consumed.append(kwargs['stdin'].read())
            return response
        database.run.side_effect = run
        historian.restore(database, self.archive, 'recovered')
        self.assertEqual(consumed, [self.archive.read_bytes(), self.archive.read_bytes()])
        tool, flags = database.run.call_args.args
        self.assertEqual(tool, 'pg_restore')
        self.assertIn('--single-transaction', flags)
        self.assertIn('--exit-on-error', flags)
        self.assertNotIn('--clean', flags)

    def test_restore_refuses_wrong_target_or_existing_objects(self):
        self.archive.write_bytes(b'PGDMParchive')
        for target, message in [({'database': 'other', 'objects': 0}, 'does not match'),
                                ({'database': 'recovered', 'objects': 1}, 'empty target')]:
            database = mock.Mock()
            database.run.return_value = subprocess.CompletedProcess([], 0, json.dumps(target).encode(), b'')
            with self.assertRaisesRegex(historian.HistorianError, message):
                historian.restore(database, self.archive, 'recovered')
            self.assertEqual(database.run.call_count, 2)
            self.assertEqual(database.run.call_args.args[0], 'psql')

    def test_corrupt_archive_does_not_touch_database(self):
        self.archive.write_bytes(b'SELECT bad')
        database = mock.Mock()
        with self.assertRaises(historian.HistorianError):
            historian.restore(database, self.archive, 'recovered')
        database.run.assert_not_called()

    def test_password_stays_in_environment_and_errors_are_redacted(self):
        with mock.patch.dict(os.environ, {'PGDATABASE': 'demo', 'PGPASSWORD': 'private-test-value'}, clear=True):
            database = historian.DatabaseCommands(None)
        with mock.patch('historian.subprocess.run') as run:
            run.return_value = subprocess.CompletedProcess([], 1, b'', b'failure private-test-value')
            with self.assertRaises(historian.HistorianError) as failure:
                database.run('pg_dump', ['--no-password'])
            self.assertNotIn('private-test-value', str(failure.exception))
            self.assertNotIn('private-test-value', ' '.join(run.call_args.args[0]))
            self.assertEqual(run.call_args.kwargs['env']['PGPASSWORD'], 'private-test-value')

    def test_compose_command_uses_no_tty_and_never_substitutes_secrets(self):
        compose = self.directory / 'demo compose.yml'
        compose.write_text('services: {}', encoding='utf-8')
        database = historian.DatabaseCommands(compose)
        args = database.command('pg_dump', ['--format=custom'])
        self.assertIn('-T', args)
        self.assertIn(str(compose), args)
        self.assertIn('--format=custom', args)
        self.assertIn('$POSTGRES_PASSWORD', args[args.index('-c') + 1].replace('${POSTGRES_PASSWORD:-}', '$POSTGRES_PASSWORD'))
        with mock.patch.dict(os.environ, {'PGDATABASE': 'postgresql://user:secret@host/db'}, clear=True):
            with self.assertRaisesRegex(historian.HistorianError, 'plain database name'):
                historian.DatabaseCommands(None)
        selected = historian.DatabaseCommands(compose, database_name='recovered')
        self.assertIn('OTIO_HISTORIAN_DATABASE=recovered', selected.command('pg_restore', ['--single-transaction']))

    def test_cli_failure_is_nonzero_and_restore_requires_confirmation(self):
        with mock.patch('sys.stderr', new_callable=io.StringIO), mock.patch.dict(os.environ, {}, clear=True):
            self.assertEqual(historian.main(['backup', '--native', '--output', str(self.archive)]), 1)
            with self.assertRaises(SystemExit) as failure:
                historian.main(['restore', '--native', '--input', str(self.archive)])
            self.assertEqual(failure.exception.code, 2)


@unittest.skipUnless(os.environ.get('OTIO_HISTORIAN_TEST_COMPOSE') or os.environ.get('OTIO_HISTORIAN_TEST_NATIVE') == '1',
                     'Set OTIO_HISTORIAN_TEST_COMPOSE or OTIO_HISTORIAN_TEST_NATIVE=1 for a disposable PostgreSQL server')
class HistorianIntegrationTests(unittest.TestCase):
    def test_real_custom_dump_and_restore_preserve_rows_and_refuse_nonempty_target(self):
        compose_value = os.environ.get('OTIO_HISTORIAN_TEST_COMPOSE')
        compose = Path(compose_value) if compose_value else None
        service = os.environ.get('OTIO_HISTORIAN_TEST_SERVICE', 'postgres')
        server = historian.DatabaseCommands(compose, service)
        names = ['otio_historian_source_' + uuid.uuid4().hex[:12], 'otio_historian_restore_' + uuid.uuid4().hex[:12]]
        created = []
        try:
            for name in names:
                server.run('psql', ['-X', '--set=ON_ERROR_STOP=1', '--command=CREATE DATABASE "' + name + '"'])
                created.append(name)
            source = historian.DatabaseCommands(compose, service, names[0])
            target = historian.DatabaseCommands(compose, service, names[1])
            source.run('psql', ['-X', '--set=ON_ERROR_STOP=1', "--command=CREATE TABLE metric_events (id bigint PRIMARY KEY, value text); INSERT INTO metric_events VALUES (1, 'temperature'), (2, 'Unicode: °C');"])
            with tempfile.TemporaryDirectory() as temporary:
                archive = Path(temporary) / 'live.dump'
                historian.backup(source, archive)
                historian.restore(target, archive, names[1])
                result = target.run('psql', ['-X', '--tuples-only', '--no-align', '--command=SELECT json_agg(value ORDER BY id) FROM metric_events'])
                self.assertEqual(json.loads(result.stdout.decode('utf-8')), ['temperature', 'Unicode: °C'])
                with self.assertRaisesRegex(historian.HistorianError, 'empty target'):
                    historian.restore(target, archive, names[1])
        finally:
            for name in created:
                server.run('psql', ['-X', '--set=ON_ERROR_STOP=1', '--command=DROP DATABASE "' + name + '"'])


if __name__ == '__main__':
    unittest.main()
