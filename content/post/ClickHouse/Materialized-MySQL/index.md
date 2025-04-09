---
title: ClickHouse Materialized MySQL
description: 介紹 ClickHouse 的實驗性功能 Materialized MySQL
slug: clickhouse-materialized-mysql
date: 2024-10-17T12:00:00+08:00
categories:
   - ClickHouse
weight: 1  
---
# Materialized MySQL (實驗性)

ClickHouse 提供了 MaterializedMySQL 的 database engine 允許 ClickHouse 作為 MySQL 的 replica，透過讀取 MySQL 的 binlog 來執行 DDL、DML 到 ClickHouse 上

## MySQL server

在建立 MySQL 和 ClickHouse 的同步之前，在 MySQL Server 這邊需要有以下設定：

1. MySQL 需要開啟 gtid 模式：

    ```sql
    gtid-mode = ON
    enforce-gtid-consistency = ON
    ```

2. 建立 ClickHouse 連線所需的權限，需注意 ClickHouse 只能使用 `mysql_native_password` 的 plugin 來連線 MySQL 進行驗證：

    ```sql
    CREATE USER clickhouse@'%' IDENTIFIED WITH mysql_native_password BY 'ClickHouse';
    GRANT RELOAD, REPLICATION SLAVE, REPLICATION CLIENT,SELECT ON *.* TO clickhouse@'%';
    ```


## ClickHouse 設定

設置以下設定

```sql
set allow_experimental_database_materialized_mysql = 1;
```

```sql
CREATE DATABASE [IF NOT EXISTS] db_name [ON CLUSTER cluster]
ENGINE = MaterializedMySQL('host:port', ['database' | database], 'user', 'password') [SETTINGS ...]
[TABLE OVERRIDE table1 (...), TABLE OVERRIDE table2 (...)]
```

- SETTINGS 提供的參數
    - max_rows_in_buffer：允許數據在內存中最大的行數，當超過此設定時數據將被 materialized，預設值為 65505
    - max_bytes_in_buffer
    - max_flush_data_time
    - max_wait_time_when_mysql_unavailable
    - allows_query_when_mysql_lost
    - materialized_mysql_tables_list

```sql

CREATE DATABASE db1_mysql 
ENGINE = MaterializedMySQL(
  'mysql-host.domain.com:3306', 
  'db1', 
  'clickhouse_user', 
  'ClickHouse_123'
);
```