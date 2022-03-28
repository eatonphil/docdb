# docdb

A simple Go document database.

## Build

Grab Go 1.18 and this repo. Inside this repo run:

```
$ go build
$ ./docdb
```

## Usage

Then in another terminal:

```
$ curl -X POST -H 'Content-Type: application/json' -d '{"name": "Kevin", "age": "45"}' http://localhost:8080/docs
{"body":{"id":"5ac64e74-58f9-4ba4-909e-1d5bf4ddcaa1"},"status":"ok"}
$ curl --get http://localhost:8080/docs --data-urlencode 'q=name:"Kevin"' | jq
{
  "body": {
    "count": 1,
    "documents": [
      {
        "body": {
          "age": "45",
          "name": "Kevin"
        },
        "id": "5ac64e74-58f9-4ba4-909e-1d5bf4ddcaa1"
      }
    ]
  },
  "status": "ok"
}
$ curl --get http://localhost:8080/docs --data-urlencode 'q=age:<50' | jq
{
  "body": {
    "count": 1,
    "documents": [
      {
        "body": {
          "age": "45",
          "name": "Kevin"
        },
        "id": "5ac64e74-58f9-4ba4-909e-1d5bf4ddcaa1"
      }
    ]
  },
  "status": "ok"
}
```
