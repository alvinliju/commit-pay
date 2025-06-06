
# 🎯 CommitPay – *Accountability With Teeth*

> **Bet on your discipline. Upload proof. Get verified. Or lose your money.**

---

## 🧠 The Philosophy

Todo lists are for cowards.
**CommitPay** is a wager-backed protocol for real commitments.

* No “someday” goals.
* No excuses.
* No second chances.

You define your task.
You choose your verifier.
You stake real money.

Then you either ship — or forfeit.

---

## 🧪 How It Works

1. **Create a Commitment**

   * Set task, deadline, amount, and a verifier’s email.

2. **Stake Your Money**

   * Pay via Razorpay. No stake = no skin in the game.

3. **Upload Proof**

   * File, photo, video — whatever proves you did the thing.

4. **Get Judged**

   * Your verifier gets notified and approves or rejects it.

5. **Result**

   * ✅ Approved? You win.
   * ❌ Rejected? You lose.

---

## 🛠️ Stack

| Layer        | Stack                           |
| ------------ | ------------------------------- |
| Backend      | Go + Gin                        |
| Payments     | Razorpay (live + test mode)     |
| Storage      | In-memory store                 |
| File Uploads | Local `./uploads` dir           |
| Emails       | (WIP – Resend/Sendgrid planned) |
| Frontend     | Coming soon – HTMX + Tailwind   |

---

## 🧪 API Reference (Testable via `curl`)

```bash
# 🧬 Health Check
curl http://localhost:8080/ping

# ✅ Create Bet
curl -X POST http://localhost:8080/api/bet/create \
  -H "Content-Type: application/json" \
  -d '{
    "bettorEmail": "me@example.com",
    "verifierEmail": "friend@example.com",
    "taskTitle": "Launch my side project",
    "deadline": "2025-06-15T23:59:59Z",
    "wagerAmount": 500
  }'

# 📥 Upload Proof
curl -X POST http://localhost:8080/api/bet/order_debug_fake123 \
  -F "proof=@/path/to/screenshot.png"

# 🔍 Get Bet Info
curl http://localhost:8080/api/bet/order_debug_fake123

# ✅ Verifier Approves
curl -X POST http://localhost:8080/api/verify/order_debug_fake123 \
  -H "Content-Type: application/json" \
  -d '{"approved": true}'

# 🧾 Simulate Payment Webhook
curl -X POST http://localhost:8080/api/razorpay/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "razorpay_order_id": "order_debug_fake123",
    "razorpay_payment_id": "test_pay_123",
    "razorpay_signature": "fake_sig"
  }'
```

---

## 📌 TODO (as of current state)

### 🔧 Backend

* [x] In-memory bet storage
* [x] Razorpay test mode
* [x] Proof upload endpoint
* [x] Verifier decision logic
* [x] DEBUG bypasses for dev testing
* [ ] Email notifier (to verifier on proof upload)
* [ ] File cleanup / file size checks
* [ ] Razorpay webhook security (if running in prod)
* [ ] Migrate to DB (PostgreSQL or MongoDB)

### 💻 Frontend (HTMX + Tailwind)

* [ ] Simple bet creation form
* [ ] Proof upload UI
* [ ] Verifier decision interface
* [ ] Status display (pending/submitted/approved/rejected)

### 📤 Infra

* [ ] Dockerize
* [ ] `.env.example` template
* [ ] CI for lint/test

---

## 🪪 License

MIT — use it, fork it, launch it.

---

## 🧨 Want Early Access?

Founders and high-performers testing this now.
Email: `founder@commitpay.xyz`

---
