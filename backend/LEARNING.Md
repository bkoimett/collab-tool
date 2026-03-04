# database permissions

```sql
--- connect as superuser
sudo -u postgres psql 
--- grant all priviledges

chmod +x test.sh -- give permissions to bash file

```

## Progress

1. Websocket layer - bidirectional communication / persistanet connections / low latency
2. Hub Pattern - 
3. 

sync.rwmutex - read simultaneously and write data exclussively Lock


## processes

go run *.go - run everything

```bash
## 
## start postgreSQL
sudo systemctl start postgresql 
sudo systemctl status postgresql
sudo ss -lntp | grep 5432


## create db and user

sudo -u postgres psql
CREATE USER editor_user WITH PASSWORD 'password123';
CREATE DATABASE collab_editor OWNER editor_user;
GRANT ALL PRIVILEGES ON DATABASE collab_editor TO editor_user;
\q


```