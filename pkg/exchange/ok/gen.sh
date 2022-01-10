oapi-codegen -alias-types -generate types,client,spec docs/trade.json > api/trade/trade.gen.go
oapi-codegen -alias-types -generate types,client,spec docs/account.json > api/account/account.gen.go
oapi-codegen -alias-types -generate types,client,spec docs/market.json > api/market/market.gen.go
