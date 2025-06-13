"use client";
import { collectSegments } from "next/dist/build/segment-config/app/app-segments";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

export default function viewBet() {
  const params = useSearchParams();
  const query = params.get("id");
  const [data, setData] = useState("");
  const [isLoading, setLoading] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(`http://localhost:8080/api/bet/${query}`, {
          method: "GET",
          headers: { "Content-Type": "application/json" },
        });
        const result = await response.json();
        if (result.message != "success") {
          throw Error("Error occured");
        }
        setData({ ...result.bet });
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  console.log(data);
  console.log(data.bettorEmail);

  if (isLoading) {
    return <p>Loading...</p>;
  }

  if (error) {
    return <p>Error: {error}</p>;
  }
  return (
    <>
      <h1>Bet ID: {data.id}</h1>
      <p>Task: {data.taskTitle}</p>
      <p>Bettor: {data.bettorEmail}</p>
      <p>Status: {data.status}</p>
      {data.proofURL && (
        <img src={`http://localhost:8080${data.proofURL}`} alt="proof" />
      )}
    </>
  );
}
