package blockcache

import (
	"fmt"
	"strconv"
	"testing"

	"time"

	"github.com/iost-official/Go-IOS-Protocol/account"
	"github.com/iost-official/Go-IOS-Protocol/common"
	"github.com/iost-official/Go-IOS-Protocol/core/state"
	"github.com/iost-official/Go-IOS-Protocol/core/tx"
	"github.com/iost-official/Go-IOS-Protocol/db"
	"github.com/iost-official/Go-IOS-Protocol/vm"
	"github.com/iost-official/Go-IOS-Protocol/vm/lua"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStdTxsVerifier(t *testing.T) {
	Convey("test of txs bench", t, func() {
		dbx, err := db.DatabaseFactory("redis")
		if err != nil {
			panic(err.Error())
		}
		sdb := state.NewDatabase(dbx)
		pool := state.NewPool(sdb)
		main := lua.NewMethod(2, "main", 0, 1)
		code := `function main()
				Put("hello", "world")
				return "success"
			end`

		txs := make([]*tx.Tx, 0)

		pool.PutHM("iost", "a", state.MakeVFloat(100000))

		for j := 0; j < 100; j++ {
			lc := lua.NewContract(vm.ContractInfo{Prefix: strconv.Itoa(j), GasLimit: 10000, Price: 1, Publisher: vm.IOSTAccount("a")}, code, main)
			txx := tx.NewTx(int64(j), &lc)
			txs = append(txs, &txx)
		}

		So(len(txs), ShouldEqual, 100)
		p2, i, err := StdTxsVerifier(txs, pool)
		fmt.Println(i)
		fmt.Println("error", err.Error())
		fmt.Println(p2.GetHM("iost", "a"))
		p := p2.(*state.PoolImpl)
		count := 0
		for p != nil {
			p = p.Parent()
			count++
		}
		So(count, ShouldEqual, 2)
	})
}

func TestStdCacheVerifier(t *testing.T) {
	Convey("Test of StdCacheVerifier", t, func() {
		dbx, err := db.DatabaseFactory("redis")
		So(err, ShouldBeNil)
		sdb := state.NewDatabase(dbx)
		pool := state.NewPool(sdb)
		main := lua.NewMethod(2, "main", 0, 1)
		code := `function main()
				Put("hello", "world")
				return "success"
			end`

		pool.PutHM("iost", "a", state.MakeVFloat(100000))

		for j := 0; j < 90; j++ {
			lc := lua.NewContract(vm.ContractInfo{Prefix: strconv.Itoa(j), GasLimit: 10000, Price: 1, Publisher: vm.IOSTAccount("a")}, code, main)
			txx := tx.NewTx(int64(j), &lc)
			StdCacheVerifier(&txx, pool, &vm.Context{})
		}
		v, err := pool.GetHM("iost", "a")
		So(err, ShouldBeNil)
		So(v.EncodeString(), ShouldEqual, "f9.459100000000139e+03")
		p := pool.(*state.PoolImpl)
		count := 0
		for p != nil {
			p = p.Parent()
			count++
		}
		So(count, ShouldEqual, 1)
	})

	Convey("Test of context", t, func() {
		dbx, err := db.DatabaseFactory("redis")
		So(err, ShouldBeNil)
		sdb := state.NewDatabase(dbx)
		pool := state.NewPool(sdb)
		main := lua.NewMethod(2, "main", 0, 1)
		code := `function main()
				if (Random(0.5))
				then 
					Transfer("a", "b", 10000)
				end
	print("time:")
	print(Now())
	print("height:")
	print(Height())
	return "success"
end`

		pool.PutHM("iost", "a", state.MakeVFloat(100000))

		lc := lua.NewContract(vm.ContractInfo{Prefix: "ahaha", GasLimit: 10000, Price: 1, Publisher: vm.IOSTAccount("a")}, code, main)
		txx := tx.NewTx(1, &lc)

		ctx := vm.NewContext(vm.BaseContext())
		ctx.ParentHash = []byte{1}
		ctx.BlockHeight = 10
		ctx.Timestamp = time.Now().Unix()

		err = StdCacheVerifier(&txx, pool, ctx)

		So(err, ShouldBeNil)

		balance, err := pool.GetHM("iost", "a")
		So(err, ShouldBeNil)

		So(balance.(*state.VFloat).ToFloat64() > 90000, ShouldBeTrue)

		ctx2 := vm.NewContext(ctx)
		ctx2.ParentHash = []byte{}

		err = StdCacheVerifier(&txx, pool, ctx2)
		So(err, ShouldBeNil)
		balance, err = pool.GetHM("iost", "a")
		So(err, ShouldBeNil)
		So(balance.(*state.VFloat).ToFloat64() < 90000, ShouldBeTrue)
	})
}

func BenchmarkStdTxsVerifier(b *testing.B) {
	dbx, err := db.DatabaseFactory("redis")
	if err != nil {
		panic(err.Error())
	}
	sdb := state.NewDatabase(dbx)
	pool := state.NewPool(sdb)
	main := lua.NewMethod(2, "main", 0, 1)
	code := `function main()
				Put("hello", "world")
				return "success"
			end`

	txs := make([]*tx.Tx, 0)
	for j := 0; j < 10000; j++ {
		lc := lua.NewContract(vm.ContractInfo{Prefix: strconv.Itoa(j), GasLimit: 10000, Price: 0, Publisher: vm.IOSTAccount("a")}, code, main)
		txx := tx.NewTx(int64(j), &lc)
		txs = append(txs, &txx)
	}

	for i := 0; i < b.N; i++ {
		pool, _, _ = StdTxsVerifier(txs, pool)
	}

	fmt.Println(pool.GetHM("iost", "a"))

}

func BenchmarkStdCacheVerifier(b *testing.B) {
	dbx, err := db.DatabaseFactory("redis")
	if err != nil {
		panic(err.Error())
	}
	sdb := state.NewDatabase(dbx)
	pool := state.NewPool(sdb)
	main := lua.NewMethod(2, "main", 0, 1)
	code := `function main()
				Put("hello", "world")
				return "success"
			end`

	pool.PutHM("iost", "a", state.MakeVFloat(1000000000))

	lc := lua.NewContract(vm.ContractInfo{Prefix: "ahaha", GasLimit: 10000, Price: 1, Publisher: vm.IOSTAccount("a")}, code, main)
	txx := tx.NewTx(123, &lc)
	var p2 state.Pool
	for i := 0; i < b.N; i++ {
		p2 = pool
		b.StartTimer()
		for j := 0; j < 10000; j++ {
			p2, _, err = StdTxsVerifier([]*tx.Tx{&txx}, p2)
			if err != nil {
				fmt.Println(err)
			}
		}
		b.StopTimer()
		p := p2.(*state.PoolImpl)
		count := 0
		for p != nil {
			p = p.Parent()
			count++
		}
		fmt.Println("depth:", count)
		fmt.Println(p2.GetHM("iost", "a"))
	}

}

func TestStdCacheVerifier2(t *testing.T) {

	codeA := `--- main 一元夺宝
-- snatch treasure with 1 coin !
-- @gas_limit 100000
-- @gas_price 0.001
-- @param_cnt 0
-- @return_cnt 0
-- @publisher walleta
function main()
	Put("max_user_number", 20)
	Put("user_number", 0)
	Put("winner", "")
	Put("claimed", "false")
end--f

--- BuyCoin buy coins
-- buy some coins
-- @param_cnt 2
-- @return_cnt 1
function BuyCoin(account, buyNumber)
	if (buyNumber <= 0)
	then
	    return "buy number should be more than zero"
	end

	maxUserNumber = Get("max_user_number")
    number = Get("user_number")
	if (number >= maxUserNumber or number + buyNumber > maxUserNumber)
	then
	    return string.format("max user number exceed, only %d coins left", maxUserNumber - number)
	end

	-- print(string.format("deposit account = %s, number = %d", account, buyNumber))
	Deposit(account, buyNumber)

	win = false
	for i = 0, buyNumber - 1, 1 do
	    win = win or winAfterBuyOne(number)
	    number = number + 1
	end
	Put("user_number", number)

	if (win)
	then
	    Put("winner", account)
	end

    return "success"
end--f

--- winAfterBuyOne win after buy one
-- @param_cnt 1
-- @return_cnt 1
function winAfterBuyOne(number)
	win = Random(1 - 1.0 / (number + 1))
	return win
end--f

--- QueryWinner query winner
-- @param_cnt 0
-- @return_cnt 1
function QueryWinner()
	return Get("winner")
end--f

--- QueryClaimed query claimed
-- @param_cnt 0
-- @return_cnt 1
function QueryClaimed()
	return Get("claimed")
end--f

--- QueryUserNumber query user number 
-- @param_cnt 0
-- @return_cnt 1
function QueryUserNumber()
	return Get("user_number")
end--f

--- QueryMaxUserNumber query max user number 
-- @param_cnt 0
-- @return_cnt 1
function QueryMaxUserNumber()
	return Get("max_user_number")
end--f

--- Claim claim prize
-- @param_cnt 0
-- @return_cnt 1
function Claim()
	claimed = Get("claimed")
	if (claimed == "true")
	then
		return "price has been claimed"
	end
	number = Get("user_number")
	maxUserNumber = Get("max_user_number")
	if (number < maxUserNumber)
	then
		return string.format("game not end yet! user_number = %d, max_user_number = %d", number, maxUserNumber)
	end
	winner = Get("winner")

	Put("claimed", "true")

	Withdraw(winner, number)
	return "success"
end--f
`

	codeB := `
--- main 合约主入口
-- server1转账server2
-- @gas_limit 10000
-- @gas_price 0.001
-- @param_cnt 0
-- @return_cnt 1
function main()
	print("hello")
	Transfer("iWgLQj3VTPN4dZnomuJMMCggv22LFw4nAkA6bmrVsmCo","mSS7EdV7WvBAiv7TChww7WE3fKDkEYRcVguznbQspj4K", 10)
end--f
`

	Convey("test of a bug called 0", t, func() {
		dbx, err := db.DatabaseFactory("redis")
		So(err, ShouldBeNil)
		sdb := state.NewDatabase(dbx)
		pool := state.NewPool(sdb)
		pool.PutHM("iost", "iWgLQj3VTPN4dZnomuJMMCggv22LFw4nAkA6bmrVsmCo", state.MakeVFloat(10000))

		acc, err := account.NewAccount(common.Base58Decode("3BZ3HWs2nWucCCvLp7FRFv1K7RR3fAjjEQccf9EJrTv4"))
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		_ = codeA
		_ = codeB

		rawCode := codeA
		var contract vm.Contract
		parser, _ := lua.NewDocCommentParser(rawCode)
		contract, err = parser.Parse()
		So(err, ShouldBeNil)
		mtx := tx.NewTx(1, contract)
		stx, err := tx.SignTx(mtx, acc)
		So(err, ShouldBeNil)
		buf := stx.Encode()
		var atx tx.Tx
		err = atx.Decode(buf)
		So(err, ShouldBeNil)
		err = StdCacheVerifier(&atx, pool, vm.BaseContext())
		So(err, ShouldBeNil)
	})

}

func TestStdCacheVerifierTimeout(t *testing.T) {
	Convey("test of timeout", t, func() {
		fib := `
--- main 
-- @gas_limit 10000000
-- @gas_price 0.00001
-- @param_cnt 0
-- @return_cnt 1
function main()
  n = 100
  local function inner(m)
    if m < 2 then
      return m
    end
    return inner(m-1) + inner(m-2)
  end
  print(inner(n))
end--f
`
		workable := `
--- main 合约主入口
-- server1转账server2
-- @gas_limit 10000
-- @gas_price 0.001
-- @param_cnt 0
-- @return_cnt 1
function main()
	print("hello")
end--f
`

		dbx, err := db.DatabaseFactory("redis")
		So(err, ShouldBeNil)
		sdb := state.NewDatabase(dbx)
		pool := state.NewPool(sdb)
		pool.PutHM("iost", "a", state.MakeVFloat(1000000000))

		parser, err := lua.NewDocCommentParser(fib)
		So(err, ShouldBeNil)
		contract, err := parser.Parse()
		So(err, ShouldBeNil)
		contract.SetSender("a")
		So(err, ShouldBeNil)
		mtx := tx.NewTx(1, contract)
		err = StdCacheVerifier(&mtx, pool, vm.BaseContext())
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "time out")

		parser, err = lua.NewDocCommentParser(workable)
		So(err, ShouldBeNil)
		contract, err = parser.Parse()
		So(err, ShouldBeNil)
		contract.SetSender("a")
		So(err, ShouldBeNil)
		mtx = tx.NewTx(1, contract)
		err = StdCacheVerifier(&mtx, pool, vm.BaseContext())
		So(err, ShouldBeNil)

	})
}
