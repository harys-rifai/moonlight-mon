package main
import (
 "fmt"; "log"; "os"
 _ "github.com/jackc/pgx/v5/stdlib"
 "github.com/pocketbase/dbx"
)
func main(){
 dsn:=os.Getenv("DATABASE_URL"); if dsn==""{dsn="postgres://postgres:Password09@localhost:5006/server?sslmode=disable"}
 db,err:=dbx.Open("pgx",dsn); if err!=nil{log.Fatal(err)}; defer db.Close()
 real:="rf6pna2wr52l3wz"
 var ids []struct{ID string `db:"id"`}
 if err:=db.NewQuery("select id from systems where id like 'seed%'").All(&ids); err!=nil{log.Fatal(err)}
 upd:=0
 for _,r:=range ids{
  _,err:=db.NewQuery(`update systems set users = (select jsonb_agg(distinct x) from jsonb_array_elements_text(COALESCE(users,'[]'::jsonb) || jsonb_build_array({:u}::text)) as x) where id={:id}`).
   Bind(dbx.Params{"u":real,"id":r.ID}).Execute()
  if err!=nil{log.Fatal(err)}
  upd++
 }
 fmt.Println("updated systems", upd)
 // reassign user-owned records that have no unique(user) conflict to make them visible
 for _,tbl:=range []string{"alerts","alerts_history","quiet_hours"}{
  res,err:=db.NewQuery(`update `+`"`+tbl+`"`+` set "user"={:u} where id like 'seed%'`).Bind(dbx.Params{"u":real}).Execute()
  if err!=nil{log.Fatalf("update %s: %v",tbl,err)}
  n,_:=res.RowsAffected(); fmt.Println(tbl,n)
 }
 // user_settings and universal_tokens have unique(user); keep existing/seed-per-user rows as-is
 fmt.Println("done")
}
