package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}

type Message struct {
	BPM int
}

var (
	Blockchain []Block
	Timeout    = 10 * time.Second
)

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, BPM int) (Block, error) {
	var newBlock Block
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Timestamp = time.Now().String()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if newBlock.Index-1 != oldBlock.Index {
		return false
	}

	if newBlock.PrevHash != oldBlock.Hash {
		return false
	}

	if newBlock.Hash != calculateHash(newBlock) {
		return false
	}

	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("PORT")
	log.Printf("Listening on %s\n", httpAddr)
	s := &http.Server{
		Addr:         ":" + httpAddr,
		Handler:      mux,
		ReadTimeout:  Timeout,
		WriteTimeout: Timeout,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handlePostBlockchain).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	io.Writer.Write(w, bytes)
}

func handlePostBlockchain(w http.ResponseWriter, r *http.Request) {
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	oldBlock := Blockchain[len(Blockchain)-1]
	newBlock, err := generateBlock(oldBlock, m.BPM)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
	}
	if isBlockValid(newBlock, oldBlock) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusAccepted, newBlock)
}

func respondWithJSON(w http.ResponseWriter, _ *http.Request, _ int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Serverr Error"))
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), 0, "", ""}
		genesisBlock.Hash = calculateHash(genesisBlock)
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())
}
