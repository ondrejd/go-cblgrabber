# go-cblgrabber

Data grabber for ČBL (Czech Basketball League) from [https.//nbl.basketball](https.//nbl.basketball/zapasy). All data are stored in [SQLite](https://www.sqlite.org/) database which is used also in other projects.

## Usage

Here is self-explaining example:

```bash
# 1. starting with new database
$ ./go-cblgrabbeer -initdb -season=2022/23
# 2. continue grabbing
$ ./go-cblgrabber -season=2023/24
# 3. check database
$ ls data.db -l

.rw-r--r-- 106k ondrejd 10 úno 17:00 data.db

# 4. open database
$ sqlite3 data.db

SQLite version 3.46.1 2024-08-13 09:16:08
Enter ".help" for usage hints.
sqlite> SELECT id, name FROM teams;
1|NH Ostrava
2|BK KVIS Pardubice
3|USK Praha
4|Královští sokoli
5|PUMPA Basket Brno
6|SK Slavia Praha
7|BK Olomoucko
8|BC GEOSAN Kolín
9|BK Opava
10|ERA Basketball Nymburk
11|SLUNETA Ústí nad Labem
12|BK ARMEX ENERGY Děčín
13|Sršni Photomate Písek
sqlite> 
```

You may use `./go-cblgrabber -help` for help.

## Thanks

We are using these libraries:

- [__go-sqlite3__](https://github.com/mattn/go-sqlite3)
- [__goquery__](https://github.com/PuerkitoBio/goquery)
  