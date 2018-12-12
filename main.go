package main

/* Imports
 * 4 utility libraries for formatting, handling bytes, reading and writing JSON, and string manipulation
 * 2 specific Hyperledger Fabric specific libraries for Smart Contracts
 */
import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"log"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

// Define the Smart Contract structure
type SmartContract struct {
}

// Define the structures.  Structure tags are used by encoding/json library
type User struct {
	Address string `json:"address"`
	DPCoin int `json:"dpcoin"`
}

type Poll struct {
	Host string `json:"host"`
	Detail string `json:"detail"`
	Deadline string `json:"deadline"`
	Result string `json:"result"`
}

type VoteToken struct {
	Creator  string `json:"creator"`
	Owner  string `json:"owner"`
	PollID  string `json:"pollid"`
	Response  string `json:"response"`
}

/*
 * The Init method is called when the Smart Contract is instantiated by the blockchain network
 * Best practice is to have any Ledger initialization in separate function -- see initLedger()
 */
func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	fmt.Println("Decintralized Voting is Initialized")
	return shim.Success(nil)
}

/*
 * The Invoke method is called as a result of an application request to run the Smart Contract
 * The calling application program has also specified the particular smart contract function to be called, with arguments
 */
func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {

	// Retrieve the requested Smart Contract function and arguments
	function, args := APIstub.GetFunctionAndParameters()
	// Route to the appropriate handler function to interact with the ledger appropriately
	if function == "queryUser" { //100%
		return s.queryUser(APIstub, args)
	} else if function == "initLedger" {
		return s.initLedger(APIstub)
	} else if function == "mintTokens" {
		s.mintTokens(APIstub, args)
		return shim.Success(nil)
	} else if function == "createUser" {
		return s.createUser(APIstub, args)
	} else if function == "deleteUser" {
		return s.deleteUser(APIstub, args)
	} else if function == "queryData" {
		return s.queryData(APIstub, args)
	} else if function == "queryVoteToken" {
		return s.queryVoteToken(APIstub, args)
	} else if function == "queryUserToken" {
		return s.queryUserToken(APIstub, args)
	} else if function == "createPoll" {
		return s.createPoll(APIstub, args)
	} else if function == "vote" {
		return s.vote(APIstub, args)
	} else if function == "completePoll" {
		return s.completePoll(APIstub, args)
	}

	return shim.Error("Invalid Smart Contract function name.")
}

/*
 * Display one user.
 * Args": userID -> ["USER1"]
 */
func (s *SmartContract) queryUser(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	userAsBytes, _ := APIstub.GetState(args[0])
	return shim.Success(userAsBytes)
}

/*
 * Populate the ledger with sample users
 * Args": none
 */
func (s *SmartContract) initLedger(APIstub shim.ChaincodeStubInterface) sc.Response {
	fmt.Println("Running initLedger")
	users := []User{
		User{Address: "kirby", DPCoin: 100},
		User{Address: "ribky",DPCoin: 100},
		User{Address: "kyrib",DPCoin: 100},
		User{Address: "bikry",DPCoin: 100},
		User{Address: "irkby",DPCoin: 100},
	}

	i := 0
	for i < len(users) {
		fmt.Println("i is ", i)
		userAsBytes, _ := json.Marshal(users[i])
		APIstub.PutState("USER"+strconv.Itoa(i), userAsBytes)
		fmt.Println("Added", users[i])
		i = i + 1
	}
	return shim.Success(nil)
}


/*
 * Create votetokens for distribution to voters
 * Args": creater, CSV of addresses , pollID
 */
func (s *SmartContract) mintTokens(APIstub shim.ChaincodeStubInterface, args []string) map[string]string {
	if len(args) != 3 {
		fmt.Println("Incorrect number of arguments. Expecting 3")
		return nil
	}

	startKey := "VOTETOKEN0"
	endKey := "VOTETOKEN999"

	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	defer resultsIterator.Close()

	var keystr string;
	var maxindex int;
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			fmt.Println(err)
			return nil
		}
		fmt.Println(string(queryResponse.Key))
		keystr = string(queryResponse.Key)

		lastkey := strings.Split(keystr, "VOTETOKEN")[1]
		newindex, err := strconv.Atoi(lastkey)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		if maxindex < newindex{
			maxindex = newindex
		}
	}
	
	maxindex = maxindex + 1;
	addresses := strings.Split(args[1], ",")

	// create a map of user addresses and votetoken ids
	var usrvt map[string]string
	usrvt = make(map[string]string)

	i := 0;
	for i < len(addresses){
		address := strings.TrimSpace(addresses[i]);
		votetoken := VoteToken{}
		votetoken.Creator = args[0];
		votetoken.Owner = address;
		votetoken.PollID = args[2];
		votetoken.Response = "";
		votetokenAsBytes, _ := json.Marshal(votetoken)
		vtid := "VOTETOKEN"+strconv.Itoa(maxindex + i)
		APIstub.PutState(vtid, votetokenAsBytes)
		fmt.Println("Added", votetoken)
		usrvt[address] = vtid
		i = i + 1
	}
	fmt.Println("mintToken finished")
	return usrvt
}

/*
 * Take in a list of voters. Create a poll by minting an identical VoteToken for each user 
 * to invite each of them. Locks the owner from setting up another poll.
 * Args": owner of vote, CSV of addresses of users to vote, pollID
 */
 func (s *SmartContract) createPoll(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}
	
	//find a spot to put the new poll
	startKey := "POLL0"
	endKey := "POLL999"

	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	var keystr string;
	var maxindex int;
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		fmt.Println(string(queryResponse.Key))
		keystr = string(queryResponse.Key)

		lastkey := strings.Split(keystr, "POLL")[1]
		newindex, err := strconv.Atoi(lastkey)
		if err != nil {
			return shim.Error(err.Error())
		}
		if maxindex < newindex{
			maxindex = newindex
		}
	}
	
	maxindex = maxindex + 1
	newpoll := Poll{}
	newpoll.Host = args[0]
	newpoll.Detail = args[2]
	newpoll.Deadline = args[3]
	newpoll.Result = ""
	pollAsBytes, _ := json.Marshal(newpoll)
	newpollid := "POLL"+strconv.Itoa(maxindex)
	APIstub.PutState(newpollid, pollAsBytes)


	_ = s.mintTokens(APIstub, []string{args[0], args[1], newpollid})
	//addresses := strings.Split(args[1], ",")
	fmt.Println("Poll created")
	return shim.Success(nil)
 }

 /*
 func (s *SmartContract) invite(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	fmt.Println("invite started")
	fmt.Println("inviting", args[0], args[1])
	votetokenAsBytes, _ := APIstub.GetState("args[1]")
	votetoken := VoteToken{}
	json.Unmarshal(votetokenAsBytes, &votetoken)
	fmt.Println(votetoken)
	return shim.Success(nil)
 }
 */

/*
 * Create a user
 * Args": NA
 */
func (s *SmartContract) createUser(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}
	valueArg, _ := strconv.Atoi(args[2])
	var user = User{Address: args[1], DPCoin: valueArg}
	userAsBytes, _ := json.Marshal(user)
	APIstub.PutState(args[0], userAsBytes)
	fmt.Println(args[1], args[2], "is created")
	return shim.Success(nil)
}

/*
 * Create a user
 * Args": NA
 */
func (s *SmartContract) deleteUser(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	//var carID string
	var err error
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("deleteUser is running", args[0])
	err = APIstub.DelState(args[0])
    if err != nil {
        return shim.Error("DELSTATE failed! : " + fmt.Sprint(err))
    }

	return shim.Success(nil)
}

/*
 * Display the specified data on the vote platform
 * TESTING function does not work in real network
 * Args": none
 */
func (s *SmartContract) queryData(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	startKey := args[0] + "0"
	endKey := args[0] + "999"

	
	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")
		var data = string(queryResponse.Value)
		fmt.Println(data)
		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(data)
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	//fmt.Printf("- queryData:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

/*
 * Args": votetoken id
 */
func (s *SmartContract) queryVoteToken(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	votetokenAsBytes, _ := APIstub.GetState(args[0])
	return shim.Success(votetokenAsBytes)
}

/*
 * Display all tokens associated with this user
 * TESTING function does not work in real network
 * Args": none
 */
 func (s *SmartContract) queryUserToken(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	var owner = args[0] 
	tokenIds := s.getUserTokens(APIstub, owner)
	
	fmt.Printf("|%-10s|%-10s|\n", owner, tokenIds)
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"User\":")
	buffer.WriteString("\"")
	buffer.WriteString(owner)
	buffer.WriteString("\"")
	buffer.WriteString(", \"Tokens\":")
	buffer.WriteString(tokenIds)
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("- queryUserToken:\n%s\n", buffer.String())
	return shim.Success(buffer.Bytes())
}

/*
 * Display all tokens associated with this user
 * TESTING function does not work in real network
 * Args": address of the user
 * return: CSV of vote tokens
 */
func (s *SmartContract) getUserTokens(APIstub shim.ChaincodeStubInterface, arg string) (string){
	startKey := "VOTETOKEN0"
	endKey := "VOTETOKEN999"
	resultsIterator, _ := APIstub.GetStateByRange(startKey, endKey)
	defer resultsIterator.Close()
	var tokenKeys string

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, _ := resultsIterator.Next()
		// Record is a JSON object, so we write as-is
		var jsvalue = queryResponse.Value
		votetoken := VoteToken{}
		json.Unmarshal(jsvalue, &votetoken)
		owner := votetoken.Owner
		if owner == arg{
			if bArrayMemberAlreadyWritten == true{
				tokenKeys = tokenKeys + ", "
			}
			tokenKeys = tokenKeys + queryResponse.Key
		}
		bArrayMemberAlreadyWritten = true
	}
	return tokenKeys
}

/*
 * Display all tokens associated with this poll
 * TESTING function does not work in real network
 * Args": id of poll
 * return: CSV of vote tokens
 */
 func (s *SmartContract) getPollTokens(APIstub shim.ChaincodeStubInterface, arg string) (string){
	startKey := "VOTETOKEN0"
	endKey := "VOTETOKEN999"
	resultsIterator, _ := APIstub.GetStateByRange(startKey, endKey)
	defer resultsIterator.Close()
	var tokenKeys string

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, _ := resultsIterator.Next()
		// Record is a JSON object, so we write as-is
		var jsvalue = queryResponse.Value
		votetoken := VoteToken{}
		json.Unmarshal(jsvalue, &votetoken)
		pollid := votetoken.PollID
		if pollid == arg{
			if bArrayMemberAlreadyWritten == true{
				tokenKeys = tokenKeys + ", "
			}
			tokenKeys = tokenKeys + queryResponse.Key
		}
		bArrayMemberAlreadyWritten = true
	}
	return tokenKeys
}

/*
 * Modifies the response in a VoteToken, and send it back to the poll owner. 
 * The transaction will only be allowed if the receiver of the VoteToken is the creater of the token
 * Args": voter, owner of vote, pollID, response
 */
 func (s *SmartContract) vote(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}
	// array of tokens owned by this user
	tokenIds := strings.Split(s.getUserTokens(APIstub, args[0]), ",")

	i := 0
	for i < len(tokenIds){
		tokenId := strings.TrimSpace(tokenIds[i]);
		votetokenAsBytes, _ := APIstub.GetState(tokenId)
		votetoken := VoteToken{}
		json.Unmarshal(votetokenAsBytes, &votetoken)
		// if the creator of this token is the owner of the vote and the pollIDs match
		if votetoken.Creator == args[1] && votetoken.PollID == args[2]{
			votetoken.Response = args[3]
			fmt.Println("User", votetoken.Owner, "voted", votetoken.Response)
			// transfer the ownership of the votetoken back to the creator
			votetoken.Owner = args[1]
			votetokenAsBytes, _ = json.Marshal(votetoken)
			APIstub.PutState(tokenId, votetokenAsBytes)
			return shim.Success(nil)
		}
		i = i + 1
	}
	return shim.Error("Vote invitation matching the given information not found")
 }

 /*
 * Read the result by checking all received VoteToken, then these VoteTokens are destroyed. 
 * Unlock the poll owner to set up the next round. and determine the lottery winner (needs more research)
 * Args": pollID
 */
 func (s *SmartContract) completePoll(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	// array of tokens owned by this user
	tokenIds := strings.Split(s.getPollTokens(APIstub, args[0]), ",")
	var results map[string]int
	results = make(map[string]int)
	i := 0
	for i < len(tokenIds){
		tokenId := strings.TrimSpace(tokenIds[i]);
		votetokenAsBytes, _ := APIstub.GetState(tokenId)
		votetoken := VoteToken{}
		json.Unmarshal(votetokenAsBytes, &votetoken)
		fmt.Println(votetoken)
		ans := votetoken.Response
		if (ans != ""){
			if _, ok := results[ans]; ok {
				results[ans] = results[ans] + 1
			} else {
				results[ans] = 1
			}
		}
		
		// if the creator of this token is the owner of the vote and the pollIDs match
		/*
		if votetoken.Creator == args[1] && votetoken.PollID == args[2]{
			votetoken.Response = args[3]
			fmt.Println("User", votetoken.Owner, "voted", votetoken.Response)
			// transfer the ownership of the votetoken back to the creator
			votetoken.Owner = args[1]
			votetokenAsBytes, _ = json.Marshal(votetoken)
			APIstub.PutState(tokenId, votetokenAsBytes)
			return shim.Success(nil)
		}
		*/
		i = i + 1
	}
	var maxv int
	var maxk string
	for k, v := range results {
        if v > maxv{
			maxv = v
			maxk = k
		} else if v == maxv{
			maxk = maxk + ", " + k
		}
	}
	fmt.Println("winner is", maxk)

	pollAsBytes, _ := APIstub.GetState(args[0])
	poll := Poll{}
	json.Unmarshal(pollAsBytes, &poll)
	poll.Result = maxk
	pollAsBytes, _ = json.Marshal(poll)
	APIstub.PutState(args[0], pollAsBytes)

	return shim.Success(nil)
 } 

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {
	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}
