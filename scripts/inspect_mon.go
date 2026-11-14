package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 type M struct{ID string `db:"id"`; System string `db:"system"`; Target string `db:"target"`; Proto string `db:"protocol"`; Port int `db:"port"`; Server string `db:"server"`; Enabled bool `db:"enabled"`}
 var ms []M
 if err:=db.NewQuery(`select id,system,target,protocol,port,server,enabled from network_monitors where id like 'seedmon%' limit 5`).All(&ms); err!=nil{log.Fatal(err)}
 for _,m:=range ms{ fmt.Printf("%s|%s|%s|%s|%d|%s|%v\n",m.ID,m.System,m.Target,m.Proto,m.Port,m.Server,m.Enabled) }
}
