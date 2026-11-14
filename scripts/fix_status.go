package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 res,err:=db.NewQuery(`update systems set status='up' where id like 'seed%'`).Execute(); if err!=nil{log.Fatal(err)}
 n,_:=res.RowsAffected(); fmt.Println("systems_up", n)
 res,err=db.NewQuery(`update containers set status='running', health=1 where id like 'seed%'`).Execute(); if err!=nil{log.Fatal(err)}
 n,_=res.RowsAffected(); fmt.Println("containers_running", n)
}
