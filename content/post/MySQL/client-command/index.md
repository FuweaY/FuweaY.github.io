---
title: MySQL Client Command
description: 介紹 MySQL Client 冷門的 command
slug: mysql-client-command
date: 2020-09-20T12:00:00+08:00
categories:
   - MySQL
tags:
   - 工具
weight: 1  
---

## pager

可以做到 Linux 中的 `pipe(|)`功能，只需要 client 端中輸入關鍵字 `pager`並加上 `Linux shell` 命令即可，若有結束只需要再次輸入 `pager` 即可。

1. 範例：透過 `grep -v`，在 `show processlist` 時過濾掉不需要看到的 `user`

    ```sql
    mysql> show processlist;
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    | Id | User            | Host          | db   | Command     | Time | State                                                         | Info             |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    |  1 | event_scheduler | localhost     | NULL | Daemon      |   62 | Waiting on empty queue                                        | NULL             |
    |  3 | rep             | gateway:56222 | NULL | Binlog Dump |   59 | Master has sent all binlog to slave; waiting for more updates | NULL             |
    |  4 | root            | localhost     | NULL | Query       |    0 | starting                                                      | show processlist |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    3 rows in set (0.00 sec)
    
    mysql> pager grep -v rep
    PAGER set to 'grep -v rep'
    
    mysql> show processlist;
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    | Id | User            | Host          | db   | Command     | Time | State                                                         | Info             |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    |  1 | event_scheduler | localhost     | NULL | Daemon      |  125 | Waiting on empty queue                                        | NULL             |
    |  4 | root            | localhost     | NULL | Query       |    0 | starting                                                      | show processlist |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    3 rows in set (0.01 sec)
    
    mysql> pager
    Default pager wasn't set, using stdout.
    ```

2. 範例：透過 `cat > /dev/null`，讓查詢不顯示結果集只顯示執行結果，可以用於方便只需要單純比對查詢執行時間的情境

    ```sql
    mysql> select * from test;
    +----+------+------+
    | id | name | age  |
    +----+------+------+
    |  0 | C    |   26 |
    |  1 | A    |   30 |
    |  2 | B    |   25 |
    +----+------+------+
    3 rows in set (0.00 sec)
    
    mysql> pager cat > /dev/null
    PAGER set to 'cat > /dev/null'
    
    mysql> select * from test;
    3 rows in set (0.00 sec)
    
    mysql> pager
    Default pager wasn't set, using stdout.
    ```

3. 範例：我們都知道 `show engine innodb status` 的輸出很長，這時候就可以透過 `less` 方便查看

    ```sql
    mysql> pager less
    PAGER set to 'less'
    
    mysql> show engine innodb status\G
    1 row in set (0.00 sec)
    
    mysql> pager
    Default pager wasn't set, using stdout.
    ```

4. 範例：透過 `awk` 取得 Command 結果，並透過 `sort` 排序，最後 `uniq -c`去除重複並計算數量

    ```sql
    mysql> show processlist;
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    | Id | User            | Host          | db   | Command     | Time | State                                                         | Info             |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    |  1 | event_scheduler | localhost     | NULL | Daemon      | 1024 | Waiting on empty queue                                        | NULL             |
    |  3 | rep             | gateway:56222 | NULL | Binlog Dump | 1021 | Master has sent all binlog to slave; waiting for more updates | NULL             |
    |  6 | root            | localhost     | NULL | Query       |    0 | starting                                                      | show processlist |
    +----+-----------------+---------------+------+-------------+------+---------------------------------------------------------------+------------------+
    3 rows in set (0.00 sec)
    
    mysql> pager  awk -F '|' '{print $6}' | sort | uniq -c
    PAGER set to 'awk -F '|' '{print $6}' | sort | uniq -c'
    
    mysql> show processlist;
          3
          1  Binlog Dump
          1  Command
          1  Daemon
          1  Query
    3 rows in set (0.00 sec)
    ```


## prompt

可以自定義 mysql 的 prompt，預設為 `mysql>` 。

有以下方式可以設置：

- 設定在 OS 層，可以使用 $() 來做命令替換：
    - 設定 MYSQL_PS1 環境變數

        ```sql
        export MYSQL_PS1="`hostname -s` \r:\m:\s [\d] mysql> "
        ```

    - 連線時指定

        ```sql
        mysql -u -p --prompt="`hostname -s` \r:\m:\s [\d] mysql> "
        ```

    - 透過 aliase (可以寫到 bashrc 持久化)

        ```sql
        alias mysql='mysql --prompt="`hostname -s` \r:\m:\s [\d] mysql> "'
        ```

- 設定在 my.cnf 中

    ```sql
    [mysql] (或 [client])
    prompt= "host名稱 \\r:\\m:\\s [\\d] mysql>\\_"
    ```


上述的應用結果如下：

```sql
[root@test-11 ~]$ mysql -uGuCi -p
Enter password:
test-11 01:51:17 [(none)] mysql>
```

# 參考

[MySQL :: MySQL 8.0 Reference Manual :: 4.5.1.2 mysql Client Commands](https://dev.mysql.com/doc/refman/8.0/en/mysql-commands.html)