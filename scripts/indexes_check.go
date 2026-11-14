package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 q:=`select tablename,indexname,indexdef from pg_indexes where schemaname='public' and tablename in ('systems','system_details','containers','systemd_services','smart_devices','zfs_pools','fingerprints','universal_tokens','network_monitors','network_monitor_stats','system_stats','container_stats','alerts','alerts_history','user_settings','quiet_hours','users') order by tablename,indexname`
	type R struct{Table string `db:"tablename"`; Name string `db:"indexname"`; Def string `db:"indexdef"`}
	var rs []R
	if err:=db.NewQuery(q).All(&rs); err!=nil{log.Fatal(err)}
	for _,r:=range rs{ fmt.Printf("%s | %s | %s\n", r.Table,r.Name,r.Def) }
}
