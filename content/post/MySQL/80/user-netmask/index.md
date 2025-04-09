---
title: MySQL8.0.23 user 支持 netmask 遮罩
description: 介紹 MySQL8.0.23 在創建 user 白名單時支持 netmask 
slug: mysql-user-netmask
date: 2021-02-26T12:00:00+08:00
categories:
  - MySQL
tags:
  - "8.0"
  - 功能
weight: 1  
---
MySQL 從 8.0.23 開始支持在創建 user 時使用 netmask 遮罩來限制用戶的訪問範圍。

```bash
mysql> SELECT @@version;
+-----------+
| @@version |
+-----------+
| 8.0.23    |
+-----------+
1 row in set (0.00 sec)

mysql> create user test@'172.17.0.0/24' identified by 'test';
Query OK, 0 rows affected (0.01 sec)

mysql> select user,host from mysql.user;
+------------------+---------------+
| user             | host          |
+------------------+---------------+
| test             | 172.17.0.0/24 |
...
+------------------+---------------+

HOST 172.17.0.1 >  mysql -utest -p --host 172.17.0.5
Enter password:
Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 20
Server version: 8.0.23 MySQL Community Server - GPL
```

參考
[https://dev.mysql.com/doc/refman/8.0/en/account-names.html](https://dev.mysql.com/doc/refman/8.0/en/account-names.html)