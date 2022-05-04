oapi-codegen -package trade -alias-types -generate types,client,spec docs/trade.json > api/trade/trade.gen.go
oapi-codegen -package account -alias-types -generate types,client,spec docs/account.json > api/account/account.gen.go
oapi-codegen -package market -alias-types -generate types,client,spec docs/market.json > api/market/market.gen.go
oapi-codegen -package public -alias-types -generate types,client,spec docs/public.json > api/public/public.gen.go
