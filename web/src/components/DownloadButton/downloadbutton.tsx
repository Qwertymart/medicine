"use client";
import React from "react";
import JSZip from "jszip";
import { saveAs } from "file-saver";
import Button from "../button/button";

interface DownloadMultipleImagesProps {
  images: string[];
}

const DownloadMultipleImages: React.FC<DownloadMultipleImagesProps> = ({
  images,
}) => {
  const handleDownloadAll = async () => {
    const zip = new JSZip();
    const folder = zip.folder("images");

    try {
      await Promise.all(
        images.map(async (url, index) => {
          const response = await fetch(url);
          if (!response.ok) {
            throw new Error(`Не удалось загрузить изображение: ${url}`);
          }
          const blob = await response.blob();
          const fileName = `image_${index + 1}.png`;
          folder?.file(fileName, blob);
        })
      );

      const zipBlob = await zip.generateAsync({ type: "blob" });
      saveAs(zipBlob, "images.zip");
    } catch (error) {
      console.error("Ошибка скачивания:", error);
    }
  };

  return (
    <Button
      size="large"
      type="submit"
      variant="mint"
      disabled={images.length === 0}
      onClick={handleDownloadAll}
    >
      Скачать ({images.length})
    </Button>
  );
};

export default DownloadMultipleImages;
