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
 var nm struct{N int `db:"n"`}; _=db.NewQuery("select count(*) n from network_monitors where id like 'seed%'").One(&nm)
 fmt.Println("nm_seed", nm.N)
 var u struct{ID string `db:"id"`; Email string `db:"email"`}; _=db.NewQuery("select id,email from users where email='harysrifai@gmail.com' or email like '%harys%' limit 3").All(&[]struct{ID string `db:"id"`; Email string `db:"email"`}{}) // ignore
}
