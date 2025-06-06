package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/razorpay/razorpay-go"
)

// info we need to create a bet bettor email, verifier email, task, deadline, wager amount

type createBetRequest struct {
	BettorEmail   string `json:"bettorEmail" binding:"required,email"`
	VerifierEmail string `json:"verifierEmail" binding:"required,email"`
	TaskTitle     string `json:"taskTitle" binding:"required"`
	Deadline      string `json:"deadline" binding:"required"` // ISO time
	WagerAmount   int64  `json:"wagerAmount" binding:"required"`
}

type pendingBet struct {
	BettorEmail   string
	VerifierEmail string
	TaskTitle     string
	Deadline      string
	WagerAmount   int64
	ProofURL      string // Add this - just a file path
	Status        string // Add this to track state
}

type paymentVerfication struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

// temp store for createBet handler
var tempBetStore = make(map[string]pendingBet)

// temp store for Bets
var betStore = make(map[string]pendingBet) //i know i should create another struct for this but ig it works and its temp anyways so...

var RAZORPAY_ID string
var RAZORPAY_SECRET string

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	RAZORPAY_ID = os.Getenv("RAZORPAY_ID")
	RAZORPAY_SECRET = os.Getenv("RAZORPAY_SECRET")

	// health checker shit
	router.GET("/ping", fuckit)

	// Create a new bet and Razorpay order
	router.POST("/api/bet/create", createBet)

	//find a bet by id and the bet id will be send to the user by email
	router.GET("/api/bet/:id", findBet)

	//route to submit proof
	router.POST("/api/bet/:id", uploadProof)

	router.POST("/api/verify/:id", verifyProof)

	// verify payment with client-side callback
	router.POST("/api/razorpay/webhook", paymentVerification)

	router.Run() // listen and serve on 0.0.0.0:8080
}

func fuckit(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}

func createBet(c *gin.Context) {
	//get the json data
	var req createBetRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	fmt.Print(req)

	betID := uuid.NewString()
	amount := req.WagerAmount * 100

	//call executeRazorpay function and create an id
	rzp_orderId, err := executerazorpay(amount, betID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error creating razorpay order id, try again later"})
		return
	}

	tempBetStore[rzp_orderId] = pendingBet{
		BettorEmail:   req.BettorEmail,
		VerifierEmail: req.VerifierEmail,
		TaskTitle:     req.TaskTitle,
		Deadline:      req.Deadline,
		WagerAmount:   req.WagerAmount * 100,
		Status:        "pending",
	}

	//send Id to frontend
	c.JSON(http.StatusOK, gin.H{"message": "success", "order_id": rzp_orderId, "betId": betID, "amount": amount})
}

func paymentVerification(c *gin.Context) {
	var req paymentVerfication

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	if err := RazorPaymentVerification(req.RazorpaySignature, req.RazorpayOrderID, req.RazorpayPaymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error verifiying payment, if money got detucted please contact the support mail."})
		return
	}

	pb, ok := tempBetStore[req.RazorpayOrderID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "No pending bet for this order ID"})
		return
	}

	betStore[req.RazorpayOrderID] = pendingBet{
		BettorEmail:   pb.BettorEmail,
		VerifierEmail: pb.VerifierEmail,
		TaskTitle:     pb.TaskTitle,
		Deadline:      pb.Deadline,
		WagerAmount:   pb.WagerAmount,
	}

	delete(tempBetStore, req.RazorpayOrderID)

	fmt.Println(betStore[req.RazorpayOrderID])

	c.JSON(http.StatusOK, gin.H{"message": "created the bet successfully, please wait for conformation email."})
}

func findBet(c *gin.Context) {

	betID := c.Param("id")

	bet, ok := betStore[betID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "cannot find the bet"})
		return
	}

	c.JSON(http.StatusFound, gin.H{"message": "success", "bet": bet})

}

func uploadProof(c *gin.Context) {
	betID := c.Param("id")

	bet, exists := betStore[betID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bet not found"})
		return
	}

	file, err := c.FormFile("proof")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	filename := fmt.Sprintf("%s_%s", betID, file.Filename)
	filepath := fmt.Sprintf("./uploads/%s", filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	bet.ProofURL = filepath
	bet.Status = "proof_submitted"

	betStore[betID] = bet

	//TODO:send email to verifier

	c.JSON(http.StatusOK, gin.H{"message": "Proof uploaded successfully"})
}

func verifyProof(c *gin.Context) {
	betID := c.Param("id")
	var payload struct {
		Approved bool `json:"approved"`
	}

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	bet, exists := betStore[betID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bet not found"})
		return
	}

	if bet.Status != "proof_submitted" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proof not submitted"})
		return
	}

	fmt.Println(payload)

	if payload.Approved {
		bet.Status = "approved"
	} else {
		bet.Status = "rejected"
	}

	betStore[betID] = bet
	c.JSON(http.StatusOK, gin.H{"message": "Verifier decision recorded", "status": bet.Status})
}

// helper functions
// custom razorpay function which returns the id
func executerazorpay(amount int64, betID string) (string, error) {

	if os.Getenv("DEBUG") == "true" {
		return "order_debug_fake123", nil
	}

	client := razorpay.NewClient(RAZORPAY_ID, RAZORPAY_SECRET)

	data := map[string]interface{}{
		"amount":   amount,
		"currency": "INR",
		"receipt":  betID,
	}

	body, err := client.Order.Create(data, nil)
	if err != nil {
		return "", errors.New("Payment not initiated")
	}
	razorId, _ := body["id"].(string)
	return razorId, nil
}

// custom razorpay function to verify if the payment has been done or not
// we should get orderId, paymentId, signature form user and verify
func RazorPaymentVerification(sign, orderId, paymentId string) error {

	if os.Getenv("DEBUG") == "true" {
		fmt.Println("DEBUG: Skipping rzp signature verification ")
		return nil
	}

	signature := sign
	secret := RAZORPAY_SECRET
	data := orderId + "|" + paymentId

	h := hmac.New(sha256.New, []byte(secret))

	_, err := h.Write([]byte(data))
	if err != nil {
		return err
	}

	sha := hex.EncodeToString(h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sha), []byte(signature)) != 1 {
		return errors.New("Payment failed")
	} else {
		return nil
	}
}
