package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 type M struct{ID,Target,Proto,Server,System string `db:"id" t:"target" p:"protocol" s:"server" sys:"system"`; Port int `db:"port"`}
 var ms []M
 err=db.NewQuery(`select id,target,protocol,port,server,system from network_monitors where port in (5006,5432,6379) or target like '%localhost%' or target like '%127.0.0.1%' limit 20`).All(&ms)
 if err!=nil{log.Fatal(err)}
 for _,m:=range ms{ fmt.Printf("%s|%s|%s|%d|%s|%s\n",m.ID,m.Target,m.Proto,m.Port,m.Server,m.System) }
 fmt.Println("---")
 type S struct{ID,Name,Host string `db:"id" n:"name" h:"host"`}
 var ss []S
 _=db.NewQuery(`select id,name,host from systems where host in ('localhost','127.0.0.1') or name like '%local%' limit 10`).All(&ss)
 for _,s:=range ss{ fmt.Printf("%s|%s|%s\n",s.ID,s.Name,s.Host) }
}
