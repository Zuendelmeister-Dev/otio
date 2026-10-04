#!/usr/bin/env python3
"""Portable PostgreSQL historian backup/restore; Python 3.10+ standard library."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

if sys.version_info < (3, 10):
    print("Install Python 3.10 or newer to use the historian backup/restore tool.", file=sys.stderr)
    raise SystemExit(1)


class HistorianError(Exception):
    pass


COMPOSE_ENV = ('export PGUSER="${POSTGRES_USER:-postgres}" '
               'PGDATABASE="${OTIO_HISTORIAN_DATABASE:-${POSTGRES_DB:-${POSTGRES_USER:-postgres}}}" '
               'PGPASSWORD="${POSTGRES_PASSWORD:-}" PGCLIENTENCODING=UTF8; ')
TARGET_QUERY = """SELECT json_build_object(
 'database', current_database(),
 'objects', (
  SELECT count(*) FROM (
   SELECT c.oid FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
    WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
   UNION ALL
   SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
   UNION ALL
   SELECT t.oid FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
    WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
   UNION ALL
   SELECT oid FROM pg_namespace WHERE nspname !~ '^pg_'
    AND nspname NOT IN ('public', 'information_schema')
   UNION ALL
   SELECT oid FROM pg_extension WHERE extname <> 'plpgsql'
  ) objects
 ));"""


def require_archive(path: Path) -> None:
    try:
        with path.open('rb') as archive:
            magic = archive.read(5)
    except OSError as exc:
        raise HistorianError(f"Cannot read backup file: {exc.strerror}") from exc
    if magic != b'PGDMP':
        raise HistorianError("Expected a PostgreSQL custom-format backup (PGDMP). Use this tool's backup command.")


class DatabaseCommands:
    def __init__(self, compose: Path | None, service: str = 'postgres', database_name: str | None = None):
        self.compose, self.service = compose, service
        self.database_name = database_name
        self.env = os.environ.copy()
        self.env.setdefault('PGCONNECT_TIMEOUT', '10')
        self.env['PGCLIENTENCODING'] = 'UTF8'
        if database_name is not None:
            if not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_.-]*', database_name):
                raise HistorianError("--database must be a plain database name.")
            self.env['PGDATABASE'] = database_name
        if compose is not None:
            if not compose.is_file():
                raise HistorianError("Compose file does not exist.")
            if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]*', service):
                raise HistorianError("Invalid Compose service name.")
        else:
            database = self.env.get('PGDATABASE', '')
            if not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_.-]*', database):
                raise HistorianError("Set PGDATABASE to a plain database name for native mode. Use PGHOST, PGPORT, PGUSER and PGPASSFILE/PGPASSWORD for connection settings.")

    def command(self, tool: str, flags: list[str], connects: bool = True) -> list[str]:
        if self.compose is not None:
            script = COMPOSE_ENV + ('exec "$@" --dbname="$PGDATABASE"' if connects else 'exec "$@"')
            selection = ['-e', 'OTIO_HISTORIAN_DATABASE=' + self.database_name] if self.database_name else []
            return ['docker', 'compose', '-f', str(self.compose), 'exec', '-T', *selection, self.service,
                    'sh', '-c', script, 'otio-historian', tool, *flags]
        args = [tool, *flags]
        if connects:
            args.append('--dbname=' + self.env['PGDATABASE'])
        return args

    def run(self, tool: str, flags: list[str], *, stdin=None, stdout=None, connects: bool = True):
        try:
            result = subprocess.run(self.command(tool, flags, connects), stdin=stdin,
                                    stdout=stdout if stdout is not None else subprocess.PIPE,
                                    stderr=subprocess.PIPE, env=self.env, timeout=900, check=False)
        except FileNotFoundError as exc:
            raise HistorianError("Required executable is missing. Install Docker Compose for container mode, or PostgreSQL client tools for native mode.") from exc
        except subprocess.TimeoutExpired as exc:
            raise HistorianError(f"{tool} exceeded the 15-minute operation limit. Verify the database before retrying.") from exc
        if result.returncode != 0:
            detail = result.stderr.decode('utf-8', errors='replace').strip()
            for name in ('PGPASSWORD', 'POSTGRES_PASSWORD'):
                password = self.env.get(name)
                if password:
                    detail = detail.replace(password, '[redacted]')
            raise HistorianError(f"{tool} failed: {detail[:2000] or 'no diagnostic output'}")
        return result


def backup(database: DatabaseCommands, output: Path) -> None:
    if output.exists():
        raise HistorianError("Backup file already exists. Choose a new name; existing backups are never overwritten.")
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(prefix='.historian-', suffix='.tmp', dir=output.parent, delete=False) as file:
            temporary = Path(file.name)
            database.run('pg_dump', ['--format=custom', '--no-owner', '--no-acl', '--no-password',
                                     '--lock-wait-timeout=5s'], stdout=file)
            file.flush()
            os.fsync(file.fileno())
        require_archive(temporary)
        # Hard-link publication is atomic and refuses to replace an existing file
        # on both Windows and POSIX, including a competing backup writer.
        os.link(temporary, output)
    except FileExistsError as exc:
        raise HistorianError("Backup file appeared during the operation; it was not overwritten.") from exc
    except OSError as exc:
        raise HistorianError(f"Cannot save backup on this filesystem: {exc.strerror}. Use a local filesystem with hard-link support.") from exc
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def restore(database: DatabaseCommands, archive: Path, confirm_database: str) -> None:
    require_archive(archive)
    # Validate the archive with the installed client before touching the target.
    with archive.open('rb') as file:
        database.run('pg_restore', ['--list'], stdin=file, stdout=subprocess.DEVNULL, connects=False)
    response = database.run('psql', ['-X', '--no-password', '--tuples-only', '--no-align',
                                    '--set=ON_ERROR_STOP=1', '--command=' + TARGET_QUERY])
    try:
        target = json.loads(response.stdout.decode('utf-8'))
        name, objects = target['database'], int(target['objects'])
    except (ValueError, KeyError, TypeError) as exc:
        raise HistorianError("Cannot verify the restore target. No restore was started.") from exc
    if not confirm_database or confirm_database != name:
        raise HistorianError("Restore target does not match --confirm-database. No restore was started.")
    if objects != 0:
        raise HistorianError("Restore requires an empty target database. Stop Lense and create a separate empty database; existing tables are never dropped by this tool.")
    with archive.open('rb') as file:
        database.run('pg_restore', ['--no-password', '--no-owner', '--no-acl', '--exit-on-error',
                                  '--single-transaction'], stdin=file)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='operation', required=True)
    for operation in ('backup', 'restore'):
        command = commands.add_parser(operation)
        mode = command.add_mutually_exclusive_group(required=True)
        mode.add_argument('--compose', type=Path, help='Compose YAML path; database service must already be running')
        mode.add_argument('--native', action='store_true', help='Use PostgreSQL tools on PATH and standard PG* environment variables')
        command.add_argument('--service', default='postgres', help='Compose database service name (default: postgres)')
        command.add_argument('--database', help='Select another plain database name, for example an empty recovery database')
        if operation == 'backup':
            command.add_argument('--output', type=Path, required=True, help='New .dump backup file (never overwritten)')
        else:
            command.add_argument('--input', type=Path, required=True)
            command.add_argument('--confirm-database', required=True, help='Exact name of the empty restore target')
    args = parser.parse_args(argv)
    try:
        database = DatabaseCommands(args.compose, args.service, args.database)
        if args.operation == 'backup':
            backup(database, args.output)
            print(f"Backup saved: {args.output}")
        else:
            restore(database, args.input, args.confirm_database)
            print("Restore completed in one transaction. Point Lense at this database and verify recent measurements before resuming normal use.")
    except (HistorianError, OSError) as exc:
        print(f"Historian {args.operation} failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
