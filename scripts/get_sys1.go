package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 type S struct{ID,Name,Host string `db:"id"`; Users string `db:"users"`}
 var s S
 err=db.NewQuery(`select id,name,host,users::text as users from systems where id='seedsys0001'`).One(&s)
 if err!=nil{log.Fatal(err)}
 fmt.Printf("%s|%s|%s|%s\n",s.ID,s.Name,s.Host,s.Users)
}
