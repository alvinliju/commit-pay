"use client";
import React, { Suspense, useState } from "react";
import { useRouter } from "next/navigation";
import { Progress } from "@/components/ui/progress";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface PaymentButtonProps {
  orderData: {
    order_id: string;
    amount: string;
    currenty: string;
  };
  betData: {
    bettorEmail: string;
    verifierEmail: string;
    taskTitle: string;
    deadline: string;
    wagerAmount: number;
  };
}

const PaymentButton = ({ orderData, betData }: PaymentButtonProps) => {
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(false);

  const makePayment = async () => {
    setIsLoading(true);

    // make an endpoint to get this key
    const key = "rzp_test_p7lg1tAmXjSZvv";
    const options = {
      key: key,
      name: "CommitPay",
      currency: orderData.currenty,
      amount: orderData.amount,
      order_id: orderData.order_id,
      modal: {
        ondismiss: function () {
          setIsLoading(false);
        },
      },
      handler: async (response: any) => {
        const verificationData = {
          razorpay_payment_id: response.razorpay_payment_id,
          razorpay_order_id: response.razorpay_order_id,
          razorpay_signature: response.razorpay_signature,
        };

        try {
          console.log(verificationData);
          const res = await fetch(
            "http://localhost:8080/api/razorpay/webhook",
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify(verificationData),
            },
          );

          if (res.ok) {
            router.push(`/success?betId=${orderData.order_id}`);
          } else {
            const data = res.json();
            console.log(data);
            alert("Payment verification failed");
          }
        } catch (error) {
          console.error("Verification error:", error);
        } finally {
          setIsLoading(false);
        }
      },
      prefill: {
        email: betData.bettorEmail,
      },
    };

    const paymentObject = new (window as any).Razorpay(options);
    paymentObject.open();

    paymentObject.on("payment.failed", function (response: any) {
      alert("Payment failed. Please try again.");
      setIsLoading(false);
    });
  };

  return (
    <>
      <Suspense fallback={<Progress />}>
        <div className="">
          <Button
            className={cn(buttonVariants({ size: "lg" }))}
            disabled={isLoading}
            onClick={() => makePayment()}
          >
            Pay Now
          </Button>
        </div>
      </Suspense>
    </>
  );
};

export default PaymentButton;
