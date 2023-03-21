# Benchmark System

Shopware benchmark system. This system does nightly benchmarks with the current trunk version of Shopware 6.

## Setup

Most servers are provisioned with NixOS which you can find in the `server` folder. 
The Ansible playbook is used to run some commands in parallel on multiple servers.
The setup consist of multiple servers:

| Name          | Type                     | Software                                          |
|---------------|--------------------------|---------------------------------------------------|
| App Server    | 3x 8 Core, 32GB memory   | Caddy, PHP 8.2 FPM                                |
| MySQL         | 3x 8 Core, 32GB memory   | MySQL 8 Primary replica                           |
| Opensearch    | 1x 8 core, 32GB memory   | OpenSearch 2.6                                    |
| Redis         | 1x 2 core, 8GB memory    | Redis 7 (object cache)                            |
| Redis Session | 1x 2 core, 8GB memory    | Redis 7 (session, cart), Rabbitmq                 |
| Locust        | 1x 16 vcore, 32GB memory | Locust                                            |
| Grafana       | 1x 2 vcore, 4GB memory   | Grafana, Loki, Prometheus, Blackfire Agent, Minio |


### Tests

All tests are running with environment variables:
    - without HTTP cache
    - 60s object cache TTL
See [Locust](https://github.com/shopware/platform/tree/trunk/src/Core/DevOps/Locust) on how to run the scenarios to your server.

### Grafana

If you are interested in the Results, you can open the [Grafana dashboard](https://grafana.sw-bench.de/d/rtrgXdxnk/locust-testruns?orgId=1).

Of course, the numbers are not comparable to the real world, but it gives us a good overview of how the application performs almost no caching.

### Relevant configs

- [PHP ini](https://github.com/shopware/shopware-benchmark/blob/main/server/app/default.nix#L11-L28)
- [PHP FPM](https://github.com/shopware/shopware-benchmark/blob/main/server/app/default.nix#L46-L55)
- [Shopware Env](https://github.com/shopware/shopware-benchmark/blob/main/roles/app/templates/.env.local.php.j2)
- [Shopware Application](https://github.com/shopware/shopware-benchmark/blob/main/roles/app/templates/shopware.yml)
- [MySQL](https://github.com/shopware/shopware-benchmark/blob/main/server/mysql/default.nix#L15-L23)
- [OpenSearch](https://github.com/shopware/shopware-benchmark/blob/main/server/elastic/default.nix#L6-L11)
- [Redis](https://github.com/shopware/shopware-benchmark/blob/main/server/redis/default.nix#L8-L20)
