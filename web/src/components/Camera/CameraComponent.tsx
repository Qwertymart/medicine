"use client";
import { useRef, useState } from "react";
import Card from "@/components/card/card";
import Button from "@/components/button/button";

interface CameraComponentProps {
  onPhotoTaken: (photoData: string) => void;
}

export default function CameraComponent({
  onPhotoTaken,
}: CameraComponentProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [isCameraActive, setIsCameraActive] = useState(false);
  const [capturedPhotos, setCapturedPhotos] = useState<string[]>([]);

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
      const photoData = canvas.toDataURL("image/png");

      setCapturedPhotos((prev) => [...prev, photoData]);

      if (onPhotoTaken) {
        onPhotoTaken(photoData);
      }
    }
  };

  const clearPhotos = () => {
    setCapturedPhotos([]);
  };

  return (
    <Card>
      <div className="camera-container">
        <h2>Камера</h2>

        <div className="camera-controls">
          {!isCameraActive ? (
            <Button size="large" variant="mint" onClick={startCamera}>
              Включить камеру
            </Button>
          ) : (
            <>
              <Button size="large" variant="mint" onClick={takePhoto}>
                Сделать фото
              </Button>
              <Button size="large" variant="mint" onClick={stopCamera}>
                Выключить камеру
              </Button>
            </>
          )}

          {capturedPhotos.length > 0 && (
            <Button size="large" variant="mint" onClick={clearPhotos}>
              Очистить фото ({capturedPhotos.length})
            </Button>
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

        {capturedPhotos.length > 0 && (
          <div className="captured-photos">
            <h3>
              Сделанные фото (автоматически попадают в буфер отправления):
            </h3>
            <div className="photos-list">
              {capturedPhotos.map((photo, index) => (
                <img
                  key={index}
                  src={photo}
                  alt={`Фото ${index + 1}`}
                  className="captured-photo"
                />
              ))}
            </div>
          </div>
        )}

        <style jsx>{`
          .camera-container {
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            text-align: center;
          }

          .camera-controls {
            margin: 20px 0;
            display: flex;
            flex-wrap: wrap;
            justify-content: center;
            gap: 10px;
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

          .captured-photos {
            margin-top: 20px;
          }

          .photos-list {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
            justify-content: center;
          }

          .captured-photo {
            width: 100px;
            height: 100px;
            object-fit: cover;
            border: 1px solid #ddd;
            border-radius: 4px;
          }
        `}</style>
      </div>
    </Card>
  );
}
