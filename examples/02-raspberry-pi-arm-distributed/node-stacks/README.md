# Node stacks

Each folder contains the Docker Compose stack copied to the matching Raspberry Pi.

- `pipresense`: Presense Modbus and Presense OPC UA simulators
- `piplc4go`: PLC4Go-style Modbus machine endpoint
- `pisense`: MQTT broker and Sense collectors
- `pilense`: Lense and the Postgres database required by Lense

Do not start these files directly on the laptop. They are copied and started by Ansible on the target nodes.
