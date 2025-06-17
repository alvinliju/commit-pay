"use client";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@radix-ui/react-label";
import { Input } from "@/components/ui/input";

export default function viewBet() {
  const params = useSearchParams();
  const query = params.get("id");
  const [data, setData] = useState("");
  const [isLoading, setLoading] = useState(false);
  const [error, setError] = useState("");

  //file Upload
  const [selectedFile, setSelectedFile] = useState(null);
  const [uploadStatus, setUploadStatus] = useState("");
  const [uploading, setUploading] = useState(false)

  const handleFileChange = (event: any) => {
    setSelectedFile(event?.target.files[0]);
  };

  const handleUpload = async () => {
    if (!selectedFile) {
      setUploadStatus("Please select a file first");
      return;
    }

    setUploading(true);
    const formData = new FormData();
    formData.append("proof", selectedFile);

    try {
      const response = await fetch(`http://localhost:8080/api/bet/${query}`, {
        method: "POST",
        body: formData,
      });

      const result = await response.json();
      
      if (response.ok) {
        setUploadStatus("✅ Proof uploaded successfully!");
        fetchData();
      } else {
        setUploadStatus(`❌ Upload failed: ${result.error || result.message}`);
      }
    } catch (err) {
      setUploadStatus(`❌ Upload failed: ${err.message}`);
    } finally {
      setUploading(false);
    }
  };

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

  useEffect(() => {
    fetchData();
  }, []);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="text-white text-xl">Loading bet...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center">
        <div className="text-red-500 text-xl">Error: {error}</div>
      </div>
    );
  }

  const getStatusColor = (status) => {
    switch(status) {
      case 'active': return 'text-blue-400 bg-blue-950/20';
      case 'proof_submitted': return 'text-yellow-400 bg-yellow-950/20';
      case 'approved': return 'text-green-400 bg-green-950/20';
      case 'rejected': return 'text-red-400 bg-red-950/20';
      default: return 'text-gray-400 bg-gray-950/20';
    }
  };

  return (
    <div className="min-h-screen bg-black text-white p-8 px-4">
      <div className="max-w-4xl mx-auto">


        {/* Main Bet Card */}
        <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6 mb-8">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div>
                <label className="text-gray-400 text-sm">Bet ID</label>
                <p className="text-white font-mono text-sm break-all">{data.id}</p>
              </div>
              
              <div>
                <label className="text-gray-400 text-sm">Task</label>
                <p className="text-white text-lg font-semibold">{data.taskTitle}</p>
              </div>
              
              <div>
                <label className="text-gray-400 text-sm">Deadline</label>
                <p className="text-white">{new Date(data.deadline).toLocaleDateString()}</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="text-gray-400 text-sm">Bettor</label>
                <p className="text-white">{data.bettorEmail}</p>
              </div>
              
              <div>
                <label className="text-gray-400 text-sm">Wager Amount</label>
                <p className="text-white text-xl font-bold">₹{data.wagerAmount / 100}</p>
              </div>
              
              <div>
                <label className="text-gray-400 text-sm">Status</label>
                <span className={`inline-block px-3 py-1 rounded-full text-sm font-medium capitalize ${getStatusColor(data.status)}`}>
                  {data.status}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Proof Section */}
        {data.proofURL && (
          <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6 mb-8">
            <h2 className="text-xl font-semibold mb-4 text-green-400">Submitted Proof</h2>
            <div className="bg-black/50 rounded-lg p-4">
              <img 
                className="max-w-full max-h-96 rounded-lg mx-auto" 
                src={`http://localhost:8080${String(data.proofURL).replace("./uploads/", "/uploads/")}`} 
                alt="Proof submission" 
              />
            </div>
          </div>
        )}

        {/* Upload Section */}
        {data.status == "active" && (
          <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6">
            <div className="mb-6">
              <h2 className="text-2xl font-semibold mb-2 text-yellow-400">Upload Your Proof</h2>
              <p className="text-gray-400">Submit evidence that you've completed the task before the deadline.</p>
            </div>
            
            <div className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="picture" className="text-white font-medium">Select Proof File</Label>
                <Input 
                  id="picture" 
                  type="file" 
                  onChange={handleFileChange}
                  className="bg-black/50 border-gray-700 text-white file:bg-blue-600 file:text-white file:border-0 file:rounded-md file:px-4 file:py-2 file:mr-4"
                  accept="image/*"
                />
                {selectedFile && (
                  <p className="text-sm text-gray-400">Selected: {selectedFile.name}</p>
                )}
              </div>
              
              <Button 
                onClick={handleUpload}
                disabled={uploading || !selectedFile}
                className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-400 px-8 py-2"
              >
                {uploading ? "Uploading..." : "Upload Proof"}
              </Button>
              
              {uploadStatus && (
                <div className={`p-4 rounded-lg ${uploadStatus.includes("✅") ? "bg-green-950/20 border border-green-800 text-green-400" : "bg-red-950/20 border border-red-800 text-red-400"}`}>
                  {uploadStatus}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}