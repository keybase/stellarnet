run_recipe File.dirname(__FILE__) + "/_common_accounts.rb"

use_manual_close

create_account :usd_gateway, :master, 100
create_account :eur_gateway, :master, 100

create_account :andrew, :master, 100
create_account :bartek, :master, 100
create_account :scott, :master, 100

close_ledger

trust :scott,  :usd_gateway, "USD"
trust :bartek, :eur_gateway, "EUR"
trust :andrew, :usd_gateway, "USD"
trust :andrew, :eur_gateway, "EUR"

close_ledger

payment :usd_gateway, :andrew, ["USD", :usd_gateway, 100]
payment :eur_gateway, :andrew, ["EUR", :eur_gateway, 100]
payment :usd_gateway, :scott,  ["USD", :usd_gateway, 100]
payment :eur_gateway, :bartek, ["EUR", :eur_gateway, 100]

close_ledger

offer :andrew, {buy:["USD", :usd_gateway], with: :native}, 20, 1.1
offer :andrew, {buy: :native, with:["EUR", :eur_gateway]}, 20, 1.2

offer :andrew, {buy:["USD", :usd_gateway], with:["EUR", :eur_gateway]}, 20, 1.3

close_ledger

path_payment_strict_send :scott, :bartek, ["EUR", :eur_gateway, 1], with: ["USD", :usd_gateway, 10], path:[]
path_payment_strict_send :scott, :bartek, ["EUR", :eur_gateway, 2], with: ["USD", :usd_gateway, 12], path:[:native]

# fail:
path_payment_strict_send :scott, :bartek, ["EUR", :eur_gateway, 100], with: ["USD", :usd_gateway, 13], path:[]

close_ledger
