"use client";
import { useRef, useState } from "react";

declare global {
  interface Window {
    Module: any;
  }
}

export default function CameraComponent() {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLVideoElement>(null);
  const [isCameraActive, setIsCameraActive] = useState(false);

  const startCamera = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { width: { ideal: 1280 }, height: { ideal: 720 } },
      });

      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        setIsCameraActive(true);
      }
    } catch (error) {
      console.error("Error accessing camera:", error);
      alert("Не удалось получить доступ к камере");
    }
  };

  const stopCamera = () => {
    if (videoRef.current && videoRef.current.srcObject) {
      const stream = videoRef.current.srcObject as MediaStream;
      const tracks = stream.getTracks();

      tracks.forEach((track) => track.stop());
      videoRef.current.srcObject = null;
      setIsCameraActive(false);
    }
  };

  const takePhoto = () => {
    if (videoRef.current && canvasRef.current) {
      const video = videoRef.current;
      const canvas = canvasRef.current;
      const context = canvas.getContext("2d");

      if (!context) return;

      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;

      context.drawImage(video, 0, 0, canvas.width, canvas.height);

      const imageData = context.getImageData(0, 0, canvas.width, canvas.height);
      const data = imageData.data;

      try {
        const photoUrl = canvas.toDataURL("image/png");
        downloadPhoto(photoUrl);
      } catch (error) {
        console.error("Error processing image with WASM:", error);
      }
    }
  };

  const downloadPhoto = (dataUrl: string) => {
    const link = document.createElement("a");
    link.download = `photo-${Date.now()}.png`;
    link.href = dataUrl;
    link.click();
  };

  return (
    <div className="camera-container">
      <h2>Камера</h2>

      <div className="camera-controls">
        {!isCameraActive ? (
          <button onClick={startCamera} className="btn btn-primary">
            Включить камеру
          </button>
        ) : (
          <>
            <button onClick={takePhoto} className="btn btn-success">
              Сделать фото
            </button>
            <button onClick={stopCamera} className="btn btn-danger">
              Выключить камеру
            </button>
          </>
        )}
      </div>

      <div className="camera-preview">
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          className="camera-video"
          style={{ display: isCameraActive ? "block" : "none" }}
        />

        <canvas
          ref={canvasRef}
          className="camera-canvas"
          style={{ display: "none" }}
        />

        {!isCameraActive && (
          <div className="camera-placeholder">Камера отключена</div>
        )}
      </div>

      <style jsx>{`
        .camera-container {
          max-width: 800px;
          margin: 0 auto;
          padding: 20px;
          text-align: center;
        }

        .camera-controls {
          margin: 20px 0;
        }

        .btn {
          padding: 10px 20px;
          margin: 0 10px;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 16px;
        }

        .btn-primary {
          background-color: #0070f3;
          color: white;
        }

        .btn-success {
          background-color: #28a745;
          color: white;
        }

        .btn-danger {
          background-color: #dc3545;
          color: white;
        }

        .camera-preview {
          position: relative;
          width: 100%;
          height: 400px;
          border: 2px solid #ddd;
          border-radius: 8px;
          overflow: hidden;
          background-color: #f5f5f5;
        }

        .camera-video {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .camera-placeholder {
          display: flex;
          justify-content: center;
          align-items: center;
          height: 100%;
          color: #666;
          font-size: 18px;
        }
      `}</style>
    </div>
  );
}
