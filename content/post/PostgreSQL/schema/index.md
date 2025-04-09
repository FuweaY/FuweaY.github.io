---
title: PostgreSQL schema
description: 紀錄學習的 PostgreSQL schema 是什麼
slug: Postgre-schema
date: 2020-02-13T12:00:00+08:00
categories:
   - PostgreSQL
weight: 1  
---
發現 PG 中有一個 SCHEMA 的層級這在 MySQL 中是沒有的。

## 與 MySQL 類比

在 MySQL 中分為三個層級 `Instance` → `DATABASE` → `TABLE`，在 MySQL 中同一個 Instance 底下的任意 Database 可以存取其他 database 的 table。

在 PG 中分為四個層級 `Instance` → `DATABASE` → `SCHEMA` → `TABLE`，在 PG 中不同 Database 之間是不能存取對方 Table 的，也就是說是獨立的。

以此類比的情況下，PG 的 SCHEMA 會比較接近 MySQL 的 Database，而 PG 的 Database 會比較接近 Instance 的隔離層級。

## 細說

每一個 Database 建立後會有以下 3 個 schema：

- pg_catalog：用於儲存 pg 系統自帶的各種 metadata。
- information_schema：用於儲存提供查詢 metadata 的查詢 view，主要是為了符合 SQL 標準。此 schema 可以單獨刪除 (但不建議)。
- public：用於儲存使用者創建的 table，但一般基於安全性、管理性不建議使用。

## 參考

[postgresql - cross-database references are not implemented: - Stack Overflow](https://stackoverflow.com/questions/51784903/cross-database-references-are-not-implemented)

[database - What is the MySQL equivalent of a PostgreSQL 'schema'? - Stack Overflow](https://stackoverflow.com/questions/1925818/what-is-the-mysql-equivalent-of-a-postgresql-schema)

[在数据库中，schema、catalog分别指的是什么？ - 知乎 (zhihu.com)](https://www.zhihu.com/question/20355738)

[postgresql的database和schema的理解_Chsavvy的博客-CSDN博客](https://blog.csdn.net/weixin_44375561/article/details/119355144)

[PostgreSQL教程--逻辑结构：实例、数据库、schema、表之间的关系_postgre逻辑结构_java编程艺术的博客-CSDN博客](https://blog.csdn.net/penriver/article/details/119680114)

[postgresql - Difference between information_schema.tables and pg_tables - Stack Overflow](https://stackoverflow.com/questions/58431104/difference-between-information-schema-tables-and-pg-tables)