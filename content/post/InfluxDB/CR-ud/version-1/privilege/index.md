---
title: InfluxDB 1.X 權限管理
description: 介紹 InfluxDB 1.X 權限管理
slug: influxdb-1-CR-ud/privilege
date: 2021-01-22T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
# 權限

# 開啟驗證

1. 建立 `admin` 帳號

    ```bash
    CREATE USER <username> WITH PASSWORD '<password>' WITH ALL PRIVILEGES
    ```

2. 調整設定檔

   預設情況下是沒有開啟驗證的，需要先到設定檔`/etc/influxdb/influxdb.conf` 將 `[http]` 底下的 `auth-enabled` 測定為 `true`

    ```bash
    ...
    [http]
      auth-enabled = true
    ...
    ```

3. 重啟

    ```bash
    systemctl restart influxdb
    ```


# 登入方式

## HTTP API

當使用 `HTTP API` 時，有以下兩種登入方式。

- 查詢時，加入 `-u <username>:<password>` 進行驗證，此為推薦的方式。

```bash
curl -G http://localhost:8086/query -u root:root --data-urlencode "q=SHOW DATABASES"
```

- 查詢時，加入 `--data-urlencode "u=<username>" --data-urlencode "p=<password>"`  進行驗證。

```bash
curl -G http://localhost:8086/query --data-urlencode "u=root" --data-urlencode "p=root" --data-urlencode "q=SHOW DATABASES"
```

## CLI

當使用 `CLI` 時，有以下三種登入方式。

- 設置環境變數後，直接登入。

```bash
export INFLUX_USERNAME=root
export INFLUX_PASSWORD=root
influx
```

- 登入時使用 `-username` 和 `-password` 選項帶入帳號密碼。

```bash
influx -username root -password root
```

- 進入 `influxdb` 後，使用 `auth` 指令驗證。

```bash
[root@localhost lib]# influx
Connected to http://localhost:8086 version 1.8.3
InfluxDB shell version: 1.8.3
> auth
username: root
password:

```

# 權限管理

## 權限

總共有以下四種權限－

- ALL PRIVILEGES： `admin` 權限。
- READ ON <database_name>：對 `<database_name>` 有 `SELECT`、`SHOW` 權限。
- WRITE ON <database_name>：對 `<database_name>` 有 `INSERT` 權限。
- ALL ON <database_name>：對 `<database_name>` 有  `INSERT`、`SELECT` 和 `SHOW` 權限。

## 指令

- 顯示所有 USER

    ```sql
    SHOW USERS
    ```

- 建立 USER

  當 `<username>` 包含保留字、特殊符號或數字開頭時，必須使用 `"` 包起來。

  `<password>` 必須使用 `'` 包起來，當包含 `'` 或 換行符號時，需使用 `/` 轉譯。

    ```sql
    -- 建立 admin user
    CREATE USER "<username>" WITH PASSWORD '<password>' WITH ALL PRIVILEGES
    
    -- 建立非 admin user
    CREATE USER "<username>" WITH PASSWORD '<password>'
    ```

- 刪除 USER

    ```sql
    DROP USER <username>
    ```

- 給予權限

    ```sql
    -- 新增 admin 權限
    GRANT ALL PRIVILEGES TO <username>
    
    -- 新增 非admin 權限
    GRANT [READ,WRITE,ALL] ON <database_name> TO <username>
    ```

- 顯示 USER 權限

    ```sql
    SHOW GRANTS FOR <user_name>
    ```

- 移除權限

    ```sql
    -- 移除 admin 權限
    REVOKE ALL PRIVILEGES FROM <username>
    
    -- 移除 非admin 權限
    REVOKE [READ,WRITE,ALL] ON <database_name> FROM <username>
    ```

- 重設密碼

    ```sql
    SET PASSWORD FOR <username> = '<password>'
    ```


# 參考

[Authentication and authorization in InfluxDB - influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/administration/authentication_and_authorization/)