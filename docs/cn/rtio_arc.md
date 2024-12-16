# RTIO

```text
                 native │ could
                        │     ┌───────┐     ┌──────────┐
 ┌──────────┐ tcp/tls   │     │       ├─────► device   │
 │ device   ├───────────┼─────►       │     │ verifier │
 └──────────┘           │     │       │     └──────────┘
                        │     │ RTIO  │
 ┌──────────┐ http/https│     │       │     ┌──────────┐
 │httpclient├───────────┼─────►       │     │ device   │
 └──────────┘           │     │       ├─────► services │
  Phone/WEB/PC...       │     └───────┘     └──────────┘
```
