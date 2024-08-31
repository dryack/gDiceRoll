* exec into psql on postgres instance

`docker-compose exec postgres psql -U gdicemoniez41 -d gdiceroll`

* test hello api

`curl "http://localhost:8185/api/hello?message=TestMessage1"`

* test admin login page

`curl http://localhost:8185/admin/login`

* test logging in

`curl -v -d "username=admin&password=password" -L http://localhost:8185/admin/login`