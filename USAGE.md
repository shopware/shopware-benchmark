## Credentials

Search in the internal password manager for credentials and create a `.envrc` from that output.

Run `source .envrc` or install https://direnv.net/ to auto source it when you open this project

## Start infrastructure

The `servers.yml` defines our infrastructure in Hetzner Cloud. 
With running `benchmark-setup infra up` it starts all servers and runs the ansible playbook on them.
It is safe to run this command multiple times, as it checks that all resources.

## Stop infrastructure

`benchmark-setup infra down` deletes again anything 

## SSH Config

The ssh hosts can be generated with `benchmark-setup generate ssh`

## Generate Ansible Inventory

The `inventory.yml` can be generated with `benchmark-setup generate ansible`

## Update DNS

The dns records can be updated with `benchmark-setup infra dns`

### Clear Cache

```bash
ansible-playbook -i inventory.yml site.yml --tags clearcache
```

### Sync / Editing files

Files should be only edited in app1 server and synched using Ansible to app2 too. The FPM has opcache preload enabled and will otherwise not recognize changes

```bash
ansible-playbook -i inventory.yml site.yml --tags sync
```

### Running Commands

Commands should be executed as `caddy`. Use `www` to switch to it easily 

## Using locally an external Dump

- Go to Hetzner Console and Create a VM with a Snapshot
- Open SSH connection to Host `ssh -L 3307:127.0.0.1:3306 root@<ip>`
- Change Shopware .env to `mysql://shopware:shopware@127.0.0.1:3307/shopware`
- Don't forget to change the Sales Channel Domain in the Admin to your local domain