package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 tables:=[]string{"systems","system_details","containers","systemd_services","smart_devices","zfs_pools","fingerprints","universal_tokens","network_monitors","network_monitor_stats","system_stats","container_stats","alerts","alerts_history","user_settings","quiet_hours"}
 for _,t:=range tables{ var c struct{N int `db:"n"`}; _=db.NewQuery("select count(*) n from "+t).One(&c); fmt.Println(t,c.N) }
}
