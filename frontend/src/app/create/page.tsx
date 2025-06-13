"use client";
import { collectSegments } from "next/dist/build/segment-config/app/app-segments";
import { useState } from "react";
import PaymentButton from "@/components/user-created/Payment-Button";

const API = process.env.NEXT_PUBLIC_API_URL;

export default function Home() {
  const [res, setRes] = useState("");
  const [bettorEmail, setBettorEmail] = useState("");
  const [verifierEmail, setVerifierEmail] = useState("");
  const [taskTitle, setTaskTitle] = useState("");
  const [deadline, setDeadline] = useState("");
  const [amount, setAmount] = useState("");

  const [order, setOrder] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  async function handleCreate(e: any) {
    e.preventDefault();
    const form = new FormData(e.target);
    // const data = Object.fromEntries(form);
    try {
      setLoading(true);
      let date = new Date(deadline).toISOString();
      const res = await fetch(`${API}/bet/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bettorEmail,
          verifierEmail,
          taskTitle,
          deadline: date,
          wagerAmount: parseInt(amount),
        }),
      });

      const data = await res.json();
      if (res.ok) {
        setOrder({
          order_id: data.order_id,
          amount: data.amount,
          currency: "INR",
        });
      } else {
        console.error("Order creation failed:", data);
        alert("Failed to create payment order");
      }
    } catch (err) {
      console.error("API error:", err);
      alert("Network error");
    } finally {
      setLoading(true);
    }
  }

  return (
    <div style={{ padding: 24, fontFamily: "monospace" }}>
      <h1>CommitPay Quick Demo</h1>
      <form onSubmit={handleCreate}>
        <input
          name="bettorEmail"
          placeholder="Your email"
          value={bettorEmail}
          onChange={(e) => setBettorEmail(e.target.value)}
          required
        />
        <br />
        <input
          name="verifierEmail"
          placeholder="Verifier email"
          value={verifierEmail}
          onChange={(e) => setVerifierEmail(e.target.value)}
          required
        />
        <br />
        <input
          name="taskTitle"
          placeholder="Task title"
          value={taskTitle}
          onChange={(e) => setTaskTitle(e.target.value)}
          required
        />
        <br />
        <input
          name="deadline"
          type="datetime-local"
          value={deadline}
          onChange={(e) => setDeadline(e.target.value)}
          required
        />
        <br />
        <input
          name="wagerAmount"
          type="number"
          placeholder="Amount (INR)"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
        />
        <br />
        <button
          type="submit"
          disabled={loading}
          className={`w-full py-2 px-4 rounded-md text-white ${
            loading ? "bg-gray-400" : "bg-blue-600 hover:bg-blue-700"
          }`}
        >
          {loading ? "Creating Order..." : "Create Bet"}
        </button>
      </form>
      {order && (
        <div className="mt-8 p-4 border-t">
          <h2 className="text-xl font-semibold mb-4">Payment</h2>
          <div className="bg-gray-50 p-4 rounded-lg mb-4">
            <p className="font-medium">Order ID: {order.order_id}</p>
            <p className="font-medium">Amount: ₹{order.amount / 100}</p>
          </div>
          <PaymentButton
            orderData={order}
            betData={{
              bettorEmail,
              verifierEmail,
              taskTitle,
              deadline,
              wagerAmount: parseInt(amount),
            }}
          />
        </div>
      )}
    </div>
  );
}
