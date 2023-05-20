# gopher-rss-proxy

proxies gopher url directories as http rss feeds (rudimentary)

# usage
```
go build
./gopher-rss
```

access `http://localhost:9000/<gopher_url>` where `gopher_url` is a gopher url directory link

## with docker

```
docker build -t gopher-rss-proxy .
docker run --restart always -d -p <host port>:9000 gopher-rss-proxy
```

----

(note, some code was borrowed from https://git.mills.io/prologic/gopherproxy/src/branch/master/gopherproxy.go)
