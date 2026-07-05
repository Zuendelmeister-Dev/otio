FROM python:3.12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-client sshpass ca-certificates tar gzip \
    && rm -rf /var/lib/apt/lists/*

RUN python -m pip install --no-cache-dir ansible

COPY entrypoint.sh /usr/local/bin/otio-ansible-entrypoint
RUN chmod +x /usr/local/bin/otio-ansible-entrypoint

ENV ANSIBLE_CONFIG=/workspace/deploy/ansible/ansible.cfg

WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/otio-ansible-entrypoint"]
