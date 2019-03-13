# hlogtop

cli to parse and summarize http log in real time.

# install

```sh
go get github.com/clementauger/hlogtop
```

# usage

```sh
$ hlogtop -h
Usage of hlogtop:
  -code value
    	only those http codes (comma split)
  -cut int
    	cut some caracters from the beginning of each line
  -group value
    	url group regexp such as [name]=[re]
  -i	invert foreground print color
  -mode string
    	how to organize hits url|ua (default "url")
  -v	verbose mode
```

## example

```sh
# mode url
journalctl --no-tail -f -u [service] \
hlogtop \
-cut=56 \
-group="asset=.+\.(css|js|png|jpg|gif|ico|woff2?\?.+)$" \
-group="by_id=^/products/[0-9]+" \
-group="by_author=^/products/by_author/.+" \
-group="wp=(wp-).+"

# mode ua
journalctl --no-tail -f -u [service] \
hlogtop -mode=ua \
-cut=56 \
-group="safari=Safari" \
-group="google_bot=Googlebot" \
-group="firefox=Firefox"
```
