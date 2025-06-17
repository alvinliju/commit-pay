package main

import (
	"context"
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
	"github.com/resend/resend-go/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	BetID         string
	BettorEmail   string
	VerifierEmail string
	TaskTitle     string
	Deadline      string
	WagerAmount   int64
	ProofURL      string // Add this - just a file path
	Status        string // Add this to track state
}

type Bet struct {
	ID            string `json:"id"`
	BettorEmail   string `json:"bettorEmail"`
	VerifierEmail string `json:"verifierEmail"`
	TaskTitle     string `json:"taskTitle"`
	Deadline      string `json:"deadline"`
	WagerAmount   int64  `json:"wagerAmount"`
	ProofURL      string `json:"proofURL"`
	Status        string `json:"status"` // "pending_payment", "active", "proof_submitted", "approved", "rejected"
	CreatedAt     string `json:"createdAt"`
}

type paymentVerfication struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

// temp store for createBet handler
var tempBetStore = make(map[string]pendingBet)

// temp store for Bets
var betStore = make(map[string]Bet) //i know i should create another struct for this but ig it works and its temp anyways so...

var RAZORPAY_ID string
var RAZORPAY_SECRET string
var MONGO_URI string
var RESEND_API string

// db integration
var mongoClient *mongo.Client
var db *mongo.Database

func initMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MONGO_URI))
	if err != nil {
		log.Fatal(err)
	}
	mongoClient = client
	db = client.Database("commitpay")
}

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
	RESEND_API = os.Getenv("RESEND_API")

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

	//serving uploaded files
	router.Static("/uploads", "./uploads")

	router.Run() // listen and serve on 0.0.0.0:8080
}

func fuckit(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}

func createBet(c *gin.Context) {
	//get the json data
	var req createBetRequest
	if err := c.BindJSON(&req); err != nil {
		fmt.Println(req)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	fmt.Print(req)

	amount := req.WagerAmount * 100
	betID := uuid.NewString()

	//call executeRazorpay function and create an id
	rzp_orderId, err := executerazorpay(amount, betID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error creating razorpay order id, try again later"})
		return
	}

	tempBetStore[rzp_orderId] = pendingBet{
		BetID:         "",
		BettorEmail:   req.BettorEmail,
		VerifierEmail: req.VerifierEmail,
		TaskTitle:     req.TaskTitle,
		Deadline:      req.Deadline,
		WagerAmount:   req.WagerAmount * 100,
		Status:        "pending",
	}

	//send Id to frontend
	c.JSON(http.StatusOK, gin.H{"message": "success", "order_id": rzp_orderId, "amount": amount})
}

func paymentVerification(c *gin.Context) {
	var req paymentVerfication

	fmt.Println(req)
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

	betID := uuid.NewString()

	bet := Bet{
		ID:            betID,
		BettorEmail:   pb.BettorEmail,
		VerifierEmail: pb.VerifierEmail,
		TaskTitle:     pb.TaskTitle,
		Deadline:      pb.Deadline,
		WagerAmount:   pb.WagerAmount,
		ProofURL:      "",
		Status:        "active",
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	betStore[betID] = bet
	delete(tempBetStore, req.RazorpayOrderID)

	fmt.Println(betStore[req.RazorpayOrderID])

	bettorSubject := "Your bet has been placed!"
	bettorContent := fmt.Sprintf(`
        <h1>CommitPay Bet Confirmation</h1>
        <p>You've placed a bet of ₹%d to complete: <b>%s</b></p>
        <p>Deadline: %s</p>d
        <p>Manage your bet: <a href="http://localhost:3000/bet/?id=%s">View Bet</a></p>
    `, pb.WagerAmount/100, pb.TaskTitle, pb.Deadline, betID)

	veriferSubject := "You've been chosen as a verifier on CommitPay by your friend"
	verifierContent := fmt.Sprintf(`
			<h2>You're a Verifier! 🔍</h2>
			<p><strong>%s</strong> has bet ₹%d that they'll complete:</p>
			<h3>"%s"</h3>
			<p><strong>Deadline:</strong> %s</p>

			<p>When they submit proof, you'll get an email to verify if they actually completed the task.</p>
			<p>This is real money on the line - be honest but fair in your judgment.</p>

			<p><a href="http://localhost:3000/verify/%s">View Bet Details</a></p>
		`, bet.BettorEmail, bet.WagerAmount/100, bet.TaskTitle, bet.Deadline, bet.ID)

	go sendEmail(pb.VerifierEmail, veriferSubject, verifierContent)

	go sendEmail(pb.BettorEmail, bettorSubject, bettorContent)

	c.JSON(http.StatusOK, gin.H{"message": "created the bet successfully, please wait for conformation email."})
}

func findBet(c *gin.Context) {

	betID := c.Param("id")

	fmt.Println("id ivide", betID)

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

	approveURL := fmt.Sprintf("http://localhost:8080/api/verify/?=id%s?action=approve", bet.ID)
	rejectURL := fmt.Sprintf("http://localhost:8080/api/verify/?id=%s?action=reject", bet.ID)

	subject := fmt.Sprintf("Action Required: Verify proof for ₹%d  bet", bet.WagerAmount)
	content := fmt.Sprintf(`
			<h2>%s just submitted proof for their bet:</h2>
			<h2>The amount is wager amount is: <strong>%s</strong></h2>
			<p><strong>Deadline:</strong> %s</p>

			<p><strong>PROOF:</strong></p>
			<img src="%s" alt="proof" style="max-width: 100%%; border: 1px solid #ccc;" />
			<p>This is real money on the line - be honest but fair in your judgment.</p>
			<p>Did they actually complete this task?</p>

			<p>These buttons should work until Dec %s. </p>
			<a href=%s > accept ✅ </a>
			<a href=%s > reject ❌ </a>

		`, bet.BettorEmail, bet.TaskTitle, bet.Deadline, bet.ProofURL, bet.Deadline, approveURL, rejectURL)

	//TODO:send email to verifier
	go sendEmail(bet.VerifierEmail, subject, content)

	c.JSON(http.StatusOK, gin.H{"message": "Proof uploaded successfully"})
}

func verifyProof(c *gin.Context) {
	betID := c.Param("id")
	action := c.Param("action")

	bet, exists := betStore[betID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bet not found"})
		return
	}

	if bet.Status != "proof_submitted" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proof not submitted"})
		return
	}

	if action == "approve" {
		bet.Status = "approved"
	} else {
		bet.Status = "rejected"
	}

	betStore[betID] = bet
	c.JSON(http.StatusOK, gin.H{"message": "Verifier decision recorded", "status": bet.Status})
}

func sendEmail(email string, subject string, content string) bool {
	client := resend.NewClient(RESEND_API)

	params := &resend.SendEmailRequest{
		From:    "founder@commitpay.xyz",
		To:      []string{email},
		Html:    content,
		Subject: subject,
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		ReplyTo: "replyto@example.com",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err)
		return false
	}

	_ = sent

	return true
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
