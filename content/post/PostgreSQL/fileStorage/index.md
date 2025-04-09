---
title: PostgreSQL 實體檔案儲存
description: 紀錄學習的 PostgreSQL 實體檔案儲存
slug: Postgre-file-storage
date: 2020-02-17T12:00:00+08:00
categories:
   - PostgreSQL
weight: 1  
---
## pg_hba.conf

hba 表示 host-based authentication，該設定檔用來

- TYPE：連線方式
    - local：使用 unix domain socket 連線。
    - host：使用 TCP/IP 連線，不論是否有 SSL 加密。
    - hostssl：僅限於使用 SSL 加密的 TCP/IP 連線。
    - hostnossl：僅限於不使用 SSL 加密的 TCP/IP 連線。
- DATABASE：指定連線的 DATABASE 名稱，多個名稱可用 `,` 分隔。
    - all：表示所有 database。
    - sameuser：表示 database 名稱和 user 名稱相同。
    - samerole：表示 database 名稱和 user 所在的 role 相同。
    - replication：表示為請求 physical replication。
- USER：表示此設定針對的 USER 名稱，多個名稱可用 `,` 分隔。
    - all：表示所有 user 都適用。
- ADDRESS：表示 client 端機器 address。
- METHOD：表示驗證方式。
    - trust：無條件允許，不需要任何驗證。
    - reject：無條件拒絕。
    - scram-sha-256：使用 scram-sha-256 身分驗證，此為目前提供最安全的密碼驗證方法。
    - md5：使用 scram-sha-256 或 MD5 身分驗證，MD5 並不安全。
    - password：要求 client 端提供未加密的密碼進行身分驗證，除非使用 SSL 加密連線否則這樣是不安全的。
    - gss：使用 GSSAPI 來身分驗證，僅支援 TCP/IP 連線方式。
    - sspi：使用 SSPI 來身分驗證，僅支援 windows。

```bash
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local      database  user  auth-method  [auth-options]
host       database  user  address  auth-method  [auth-options]
hostssl    database  user  address  auth-method  [auth-options]
hostnossl  database  user  address  auth-method  [auth-options]
host       database  user  IP-address  IP-mask  auth-method  [auth-options]
hostssl    database  user  IP-address  IP-mask  auth-method  [auth-options]
hostnossl  database  user  IP-address  IP-mask  auth-method  [auth-options]
```

# 參考
[【赵渝强老师】史上最详细的PostgreSQL体系架构介绍 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/407794713)

[PostgreSQL数据目录结构 - 简书 (jianshu.com)](https://www.jianshu.com/p/cd8c5b988e52)