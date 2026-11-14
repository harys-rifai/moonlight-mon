package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pocketbase/dbx"
)

type J json.RawMessage
func (j J) Value() (interface{}, error) { if len(j)==0 {return "{}",nil}; return string(j), nil }

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { dsn = os.Getenv("POSTGRES_URL") }
	if dsn == "" { dsn = "postgres://postgres:Password09@localhost:5006/server?sslmode=disable" }
	db, err := dbx.Open("pgx", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	now := time.Now().UTC()
	ts := func(off time.Duration) string { return now.Add(off).Format("2006-01-02 15:04:05.000Z") }
	exec := func(q string, p dbx.Params) {
		if _, err := db.NewQuery(q).Bind(p).Execute(); err != nil {
			log.Fatalf("exec failed: %v\nquery: %s\nparams: %#v", err, q, p)
		}
	}
	// cleanup old seed data broadly
	for _, q := range []string{
		`delete from network_monitor_stats where id like 'seed%'`,
		`delete from network_monitors where id like 'seed%'`,
		`delete from container_stats where id like 'seed%'`,
		`delete from system_stats where id like 'seed%'`,
		`delete from alerts_history where id like 'seed%'`,
		`delete from alerts where id like 'seed%'`,
		`delete from quiet_hours where id like 'seed%'`,
		`delete from smart_devices where id like 'seed%'`,
		`delete from zfs_pools where id like 'seed%'`,
		`delete from systemd_services where id like 'seed%'`,
		`delete from containers where id like 'seed%'`,
		`delete from fingerprints where id like 'seed%'`,
		`delete from system_details where id like 'seed%' or system like 'seed%'`,
		`delete from universal_tokens where id like 'seed%' or token like 'seed%'`,
		`delete from user_settings where id like 'seed%'`,
		`delete from systems where id like 'seed%'`,
		`delete from users where id like 'seedusr%'`,
	} {
		exec(q, nil)
	}

	// create 100 seed users (auth collection requires tokenKey/password/email)
	for i := 1; i <= 100; i++ {
		id := fmt.Sprintf("seedusr%04d", i)
		email := fmt.Sprintf("seed.user%03d@moonlight.local", i)
		exec(`insert into users (id,email,"emailVisibility",verified,username,role,name,avatar,password,"tokenKey",created,updated)
			values ({:id},{:email},false,true,{:username},{:role},{:name},'',{:password},{:tokenKey},{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": id, "email": email, "username": fmt.Sprintf("seeduser%03d", i), "role": "user", "name": fmt.Sprintf("Seed User %03d", i),
				"password": "$2a$10$seedplaceholderhashseedplaceholderhashseedplaceho", "tokenKey": fmt.Sprintf("seedtokenkey%038d", i),
				"created": ts(0), "updated": ts(0)})
	}
	userIDs := make([]string, 100)
	for i := 1; i <= 100; i++ { userIDs[i-1] = fmt.Sprintf("seedusr%04d", i) }

	// 100 systems with real-ish names
	envs := []string{"prod", "staging", "dev"}
	roles := []string{"postgres", "mysql", "redis", "app", "worker", "api", "cache", "kafka", "elastic", "backup"}
	for i := 1; i <= 100; i++ {
		sid := fmt.Sprintf("seedsys%04d", i)
		role := roles[(i-1)%len(roles)]
		env := envs[(i-1)%len(envs)]
		name := fmt.Sprintf("%s-%s-%02d", env, role, i)
		host := fmt.Sprintf("10.%d.%d.%d", 10+((i-1)/64), 20+((i-1)/16)%16, 10+(i-1)%200)
		u := userIDs[i-1]
		usersJSON, _ := json.Marshal([]string{u})
		info, _ := json.Marshal(map[string]any{"h": name, "k": "6.8.0-60-generic", "c": 4 + i%8, "t": 8 + i%16, "m": "Intel Xeon/AMD EPYC", "u": 3600 * i, "cpu": float64(i%90) + 5.5, "mp": float64(30+i%60) + 0.2, "dp": float64(20+i%70) + 0.1, "os": 0, "v": "0.12.0"})
		exec(`insert into systems (id,name,status,host,port,info,users,created,updated) values ({:id},{:name},'up',{:host},'45876',{:info}::jsonb,{:users}::jsonb,{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": sid, "name": name, "host": host, "info": string(info), "users": string(usersJSON), "created": ts(0), "updated": ts(0)})
	}

	// 100 system_details (one per system)
	osNames := []string{"Ubuntu 24.04 LTS", "Debian GNU/Linux 12", "Rocky Linux 9", "AlmaLinux 9"}
	for i := 1; i <= 100; i++ {
		exec(`insert into system_details (id,system,hostname,kernel,cores,threads,cpu,os,os_name,arch,memory,podman,updated,sockets) values ({:id},{:system},{:hostname},'6.8.0-60-generic',{:cores},{:threads},{:cpu},0,{:os_name},'x86_64',{:memory},false,{:updated}::timestamp,1)`,
			dbx.Params{"id": fmt.Sprintf("seedsdt%04d", i), "system": fmt.Sprintf("seedsys%04d", i), "hostname": fmt.Sprintf("%s-host-%02d", roles[(i-1)%len(roles)], i),
				"cores": 4 + i%8, "threads": 8 + i%16, "cpu": []string{"Intel Xeon Silver 4314", "AMD EPYC 7763", "Intel Xeon E5-2686 v4", "Ampere Altra"}[i%4],
				"os_name": osNames[i%len(osNames)], "memory": int64(8+i%56) * 1024 * 1024 * 1024, "updated": ts(0)})
	}

	// 100 containers
	cnames := []string{"postgres-primary","postgres-replica","mysql-primary","redis-cache","nginx-proxy","moonlight-api","worker-jobs","kafka-broker","elasticsearch","prometheus","grafana","pgbouncer","minio","rabbitmq","vault"}
	cimages := []string{"postgres:16.4","mysql:8.4","redis:7.4-alpine","nginx:1.27-alpine","ghcr.io/moonlight/api:1.8.2","confluentinc/cp-kafka:7.7.0","docker.elastic.co/elasticsearch/elasticsearch:8.15.0","prom/prometheus:v2.54.1","grafana/grafana:11.2.0","bitnami/pgbouncer:1.23.1"}
	for i := 1; i <= 100; i++ {
		sid := fmt.Sprintf("seedsys%04d", ((i-1)%100)+1)
		name := fmt.Sprintf("%s-%02d", cnames[(i-1)%len(cnames)], i)
		exec(`insert into containers (id,system,name,status,health,cpu,memory,net,image,ports,updated,updatable) values ({:id},{:system},{:name},'running',1,{:cpu},{:memory},{:net},{:image},{:ports},{:updated},{:updatable})`,
			dbx.Params{"id": fmt.Sprintf("seedctr%04d", i), "system": sid, "name": name, "cpu": float64(i%50) + 0.5, "memory": float64(128+i%4096) * 1024 * 1024, "net": float64(1000000+i*10000), "image": cimages[(i-1)%len(cimages)], "ports": "5432/tcp,8080/tcp", "updated": now.Unix(), "updatable": i%5 == 0})
	}

	// 100 systemd_services
	svcnames := []string{"postgresql.service","mysql.service","redis-server.service","nginx.service","moonlight-api.service","kafka.service","elasticsearch.service","prometheus.service","node-exporter.service","docker.service"}
	for i := 1; i <= 100; i++ {
		exec(`insert into systemd_services (id,system,name,state,sub,cpu,"cpuPeak",memory,"memPeak",updated) values ({:id},{:system},{:name},1,0,{:cpu},{:cpuPeak},{:memory},{:memPeak},{:updated})`,
			dbx.Params{"id": fmt.Sprintf("seedsvc%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "name": fmt.Sprintf("%s", svcnames[(i-1)%len(svcnames)]), "cpu": float64(i%20) + 0.3, "cpuPeak": float64(10+i%40), "memory": float64(128+i%2048), "memPeak": float64(256+i%4096), "updated": now.Unix()})
	}

	// 100 smart_devices
	models := []string{"Samsung SSD 990 PRO 2TB","WD Black SN850X 1TB","Crucial MX500 1TB","Intel DC S4610 960GB","Seagate Exos X18 18TB"}
	for i := 1; i <= 100; i++ {
		exec(`insert into smart_devices (id,system,name,model,state,capacity,temp,firmware,serial,type,hours,cycles,attributes,updated) values ({:id},{:system},{:name},{:model},'Passed',{:capacity},{:temp},{:firmware},{:serial},{:type},{:hours},{:cycles},{:attributes}::jsonb,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedsmd%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "name": fmt.Sprintf("/dev/nvme%dn1", i%4), "model": models[(i-1)%len(models)], "capacity": float64(500+i%4000) * 1e9, "temp": 30 + i%25, "firmware": fmt.Sprintf("FW%04d", i), "serial": fmt.Sprintf("SNSEED%06d", i), "type": []string{"nvme","sata","sas"}[i%3], "hours": 100 * i, "cycles": 10 * i, "attributes": `{"Health":"OK"}`, "updated": ts(0)})
	}

	// 100 zfs_pools
	for i := 1; i <= 100; i++ {
		pname := fmt.Sprintf("tank%02d", i)
		exec(`insert into zfs_pools (id,system,name,display_name,health,status,size,alloc,free,raw,scrub,datasets,vdevs,updated,details_updated) values ({:id},{:system},{:name},{:display_name},'ONLINE','ONLINE',{:size},{:alloc},{:free},false,{:scrub}::jsonb,{:datasets}::jsonb,{:vdevs}::jsonb,{:updated}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedzfs%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "name": pname, "display_name": fmt.Sprintf("%s (data pool)", pname), "size": float64(2+i%20) * 1e12, "alloc": float64(1+i%10) * 1e12, "free": float64(1+i%10) * 1e12, "scrub": `{"state":"finished"}`, "datasets": `[]`, "vdevs": `[]`, "updated": ts(0)})
	}

	// 100 fingerprints (unique per system index exists?) and 100 universal_tokens (unique per user)
	for i := 1; i <= 100; i++ {
		exec(`insert into fingerprints (id,system,token,fingerprint,updated) values ({:id},{:system},{:token},{:fingerprint},{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedfpr%04d", i), "system": fmt.Sprintf("seedsys%04d", i), "token": fmt.Sprintf("seed-fp-token-%04d-abcdefghijkl", i), "fingerprint": fmt.Sprintf("fp:seed-system-%04d:%04x", i, i), "updated": ts(0)})
		exec(`insert into universal_tokens (id,"user",token,created) values ({:id},{:user},{:token},{:created}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedutk%04d", i), "user": fmt.Sprintf("seedusr%04d", i), "token": fmt.Sprintf("seed-universal-token-%04d", i), "created": ts(0)})
	}

	// 100 network_monitors (real DB/service names/ports)
	dbTargets := []struct{name, target, proto string; port int}{ {"PostgreSQL Primary","prod-postgres.internal","tcp",5432}, {"MySQL Reporting","mysql-reporting.internal","tcp",3306}, {"Redis Cache","redis-cache.internal","tcp",6379}, {"SQL Server DW","sqlserver-dw.internal","tcp",1433}, {"Oracle ERP","oracle-erp.internal","tcp",1521}, {"MongoDB Orders","mongodb-orders.internal","tcp",27017}, {"Elasticsearch Logs","elasticsearch.internal","tcp",9200}, {"RabbitMQ Bus","rabbitmq.internal","tcp",5672}, {"Kafka Broker","kafka-broker.internal","tcp",9092}, {"MariaDB Analytics","mariadb-analytics.internal","tcp",3306} }
	for i := 1; i <= 100; i++ {
		t := dbTargets[(i-1)%len(dbTargets)]
		exec(`insert into network_monitors (id,system,target,protocol,port,interval,enabled,res,"resAvg1h","resMin1h","resMax1h","loss1h",created,updated,"certInfo",server,"skipTlsVerify") values ({:id},{:system},{:target},{:protocol},{:port},30,true,{:res},{:avg},{:min},{:max},{:loss},{:created}::timestamp,{:updated}::timestamp,null,{:server},false)`,
			dbx.Params{"id": fmt.Sprintf("seedmon%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "target": fmt.Sprintf("%s-%02d", t.target, i), "protocol": t.proto, "port": t.port, "res": float64(i%20) + 1.1, "avg": float64(i%20) + 1.3, "min": float64(i%10) + 0.8, "max": float64(i%30) + 2.0, "loss": float64(i%3) * 0.1, "created": ts(0), "updated": ts(-time.Duration(i)*time.Minute), "server": t.name})
	}
	// 100 network_monitor_stats
	for i := 1; i <= 100; i++ {
		exec(`insert into network_monitor_stats (id,system,monitor,total_count,success_count,res_sum,res_min,res_max,type,created) values ({:id},{:system},{:monitor},{:total},{:success},{:sum},{:min},{:max},'1m',{:created})`,
			dbx.Params{"id": fmt.Sprintf("seednms%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "monitor": fmt.Sprintf("seedmon%04d", i), "total": 120, "success": 120 - i%3, "sum": 100000 + i*100, "min": 800 + i, "max": 5000 + i*10, "created": now.Unix()})
	}
	// 100 system_stats & 100 container_stats
	for i := 1; i <= 100; i++ {
		ss, _ := json.Marshal(map[string]any{"cpu": float64(i%90) + 1.0, "m": 32.0, "mu": float64(10+i%20), "mp": float64(30+i%60), "d": 500.0, "du": float64(100+i%300), "dp": float64(20+i%70), "ns": 1e6 + float64(i)*1000, "nr": 9e5 + float64(i)*800, "la": []float64{0.5, 0.6, 0.7}})
		exec(`insert into system_stats (id,system,stats,type,created,updated) values ({:id},{:system},{:stats}::jsonb,'1m',{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedsst%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "stats": string(ss), "created": ts(-time.Duration(i)*time.Minute), "updated": ts(-time.Duration(i)*time.Minute)})
		cs, _ := json.Marshal([]map[string]any{{"n": fmt.Sprintf("postgres-primary-%02d", i), "c": float64(i%50), "m": 512.0 + float64(i)}})
		exec(`insert into container_stats (id,system,stats,type,created,updated) values ({:id},{:system},{:stats}::jsonb,'1m',{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedcst%04d", i), "system": fmt.Sprintf("seedsys%04d", ((i-1)%100)+1), "stats": string(cs), "created": ts(-time.Duration(i)*time.Minute), "updated": ts(-time.Duration(i)*time.Minute)})
	}
	// 100 alerts & 100 alerts_history
	anames := []string{"CPU","Memory","Disk","Status","Bandwidth","LoadAvg1","LoadAvg5","LoadAvg15"}
	for i := 1; i <= 100; i++ {
		exec(`insert into alerts (id,"user",system,name,value,min,triggered,state,pending_since,created,updated) values ({:id},{:user},{:system},{:name},{:value},5,false,'{}'::jsonb,null,{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedalt%04d", i), "user": fmt.Sprintf("seedusr%04d", i), "system": fmt.Sprintf("seedsys%04d", i), "name": anames[(i-1)%len(anames)], "value": 70 + i%25, "created": ts(0), "updated": ts(0)})
		resolved := sql.NullString{}
		if i%2==0 { resolved = sql.NullString{String: ts(-time.Duration(i)*time.Minute), Valid: true} }
		exec(`insert into alerts_history (id,"user",system,alert_id,name,monitor_name,value,resolved,created) values ({:id},{:user},{:system},{:alert_id},{:name},{:monitor_name},{:value},{:resolved},{:created}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedhst%04d", i), "user": fmt.Sprintf("seedusr%04d", i), "system": fmt.Sprintf("seedsys%04d", i), "alert_id": fmt.Sprintf("seedalt%04d", i), "name": anames[(i-1)%len(anames)], "monitor_name": fmt.Sprintf("PostgreSQL Primary-%02d", i), "value": float64(70+i%25) + 0.5, "resolved": resolved, "created": ts(-time.Duration(i+30)*time.Minute)})
	}
	// 100 user_settings & 100 quiet_hours
	for i := 1; i <= 100; i++ {
		settings, _ := json.Marshal(map[string]any{"emails": []string{fmt.Sprintf("seed.user%03d@moonlight.local", i)}, "webhooks": []any{}, "chartTime": "1h"})
		exec(`insert into user_settings (id,"user",settings,created,updated) values ({:id},{:user},{:settings}::jsonb,{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedust%04d", i), "user": fmt.Sprintf("seedusr%04d", i), "settings": string(settings), "created": ts(0), "updated": ts(0)})
		exec(`insert into quiet_hours (id,"user",system,type,start,"end") values ({:id},{:user},{:system},{:type},{:start}::timestamp,{:end}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedqhr%04d", i), "user": fmt.Sprintf("seedusr%04d", i), "system": fmt.Sprintf("seedsys%04d", i), "type": []string{"daily","one-time"}[i%2], "start": ts(time.Duration(22+i%2)*time.Hour), "end": ts(time.Duration(30+i%2)*time.Hour)})
	}

	fmt.Println("seeded 100 rows per app table (+100 seed users)")
}
