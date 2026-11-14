package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 type M struct{ID string `db:"id"`; Target string `db:"target"`; Proto string `db:"protocol"`; Port int `db:"port"`; Server string `db:"server"`; System string `db:"system"`}
 var ms []M
 if err:=db.NewQuery(`select id,target,protocol,port,server,system from network_monitors where id like 'seedlocal%' order by id`).All(&ms); err!=nil{log.Fatal(err)}
 for _,m:=range ms{ fmt.Printf("%s | %s:%d (%s) | %s | %s\n", m.ID,m.Target,m.Port,m.Proto,m.Server,m.System) }
 type S struct{ID string `db:"id"`; Name string `db:"name"`; Host string `db:"host"`; Users string `db:"users"`}
 var s S
 if err:=db.NewQuery(`select id,name,host,users::text as users from systems where id='seedlocalsys1'`).One(&s); err!=nil{log.Fatal(err)}
 fmt.Printf("SYSTEM %s | %s | %s | %s\n", s.ID,s.Name,s.Host,s.Users)
}
