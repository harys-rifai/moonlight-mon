package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 type R struct{ID string `db:"id"`; Name string `db:"name"`; Users string `db:"users"`; Status string `db:"status"`}
 var rs []R
 _=db.NewQuery("select id,name,users::text as users,status from systems where id like 'seed%' limit 3").All(&rs)
 for _,r:=range rs{ fmt.Printf("%s|%s|%s|%s\n", r.ID,r.Name,r.Users,r.Status) }
}
