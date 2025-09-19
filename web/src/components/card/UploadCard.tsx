"use client";

import { FormEvent, useEffect, useRef, useState } from "react";
import Button from "@/components/button/button";
import Card from "@/components/card/card";
import styles from "./upload.module.css";
import Carousel from "../carousel/carousel";
import { motion, AnimatePresence } from "framer-motion";
import dynamic from "next/dynamic";
// import ResultCard from "./ResultCard";

const ResultCard = dynamic(() => import("@/components/card/ResultCard"));
const UnauthorizedBanner = dynamic(
  () => import("@/components/UnauthorizedBanner/u_banner")
);

const loader = ["https://metallsantehgroup.ru/img/load.gif"];

export default function UploadCarousel() {
  const [selectedImgs, setSelectedImgs] = useState<File[]>([]);
  const [previewUrls, setPreviewUrls] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [currentIndex, setCurrentIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const [result, setResult] = useState<string[]>([]);
  const [showResult, setShowResult] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const [unauthorized, setUnauthorized] = useState(false);

  useEffect(() => {
    return () => {
      previewUrls.forEach((url) => URL.revokeObjectURL(url));
    };
  }, [previewUrls]);

    useEffect(() => {
      const checkAuthStatus = async () => {
        try {
          const response = await fetch("/api/auth/check");
          if (!response.ok) {
            setUnauthorized(true);
            return;
          }
          setUnauthorized(false);
        } catch (error) {
          console.error("Auth check error:", error);
          setUnauthorized(true);
        }
      };

      checkAuthStatus();
    }, []);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  function handleFileChange(event: any): void {
    // ChangeEvent<HTMLInputElement>
    // const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/webp"];
    // const files: File[] = Array.from(event.target.files);

    setPreviewUrls((prev) => {
      prev.forEach((url) => URL.revokeObjectURL(url));
      return [];
    });

    if (!event.target.files || event.target.files.length === 0) {
      setSelectedImgs([]);
      setCurrentIndex(0);
      event.target.value = "";
      return;
    }

    if (event.target.files) {
      const files: File[] = Array.from(event.target.files);
      setSelectedImgs(files);

      const urls = files.map((file) => URL.createObjectURL(file));
      setPreviewUrls(urls);
      setCurrentIndex(0);
    }
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>): void {
    e.preventDefault();
    setIsLoading(true);

    const formData = new FormData();
    selectedImgs.forEach((file) => formData.append("images", file));
    console.log("fd = ", formData);
    fetch("/api/upload", {
      method: "POST",
      body: formData,
    })
      .then((response) => {
        if (!response.ok) {
          if (response.status === 401) {
            setUnauthorized(true);
          }
          throw new Error("Ошибка загрузки");
        }
        return response.json();
      })
      .then((data: UploadImagesResponse) => {
        const urls = data.results.map((result) => result.image);
        setResult(urls);
        setShowResult(true);
        setUnauthorized(false);
        setCurrentIndex(0);
      })
      .catch((err) => {
        console.error(err.message);
        return;
      })
      .finally(() => {
        setTimeout(() => {}, 5000);
        setIsLoading(false);
      });
  }

  const handleClear = () => {
    setSelectedImgs([]);
    setPreviewUrls([]);
    setCurrentIndex(0);
    setShowResult(false);
    setResult([]);

    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  return (
    <div className={styles.pageContainer}>
      {/* {unauthorized && <UnauthorizedBanner />} */}
      <UnauthorizedBanner
        message={"Необходима авторизация"}
        visible={unauthorized}
        onClose={() => setUnauthorized(false)}
      />
      <motion.div
        ref={containerRef}
        className={styles.mainContent}
        animate={{
          marginRight: showResult ? "5vw" : "0",
          transition: { duration: 0.3 },
        }}
      >
        <Card>
          <div className={styles.carouselContainer}>
            <h2 className={styles.headText}>Форма для загрузки фотографий</h2>

            {previewUrls.length > 0 && <Carousel previewUrls={previewUrls} />}

            <form onSubmit={handleSubmit} className={styles.uploadForm}>
              {previewUrls.length === 0 && (
                <div
                  className={styles.dropzone}
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.currentTarget.classList.add(styles.dragover);
                  }}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    e.currentTarget.classList.remove(styles.dragover);
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    e.currentTarget.classList.remove(styles.dragover);
                    if (e.dataTransfer.files) {
                      handleFileChange(e);
                    }
                  }}
                  onClick={() => inputRef.current?.click()}
                >
                  <input
                    type="file"
                    multiple
                    onChange={handleFileChange}
                    ref={inputRef}
                    style={{ display: "none" }}
                  />
                  <div className={styles.dropzoneContent}>
                    <p>Нажмите чтобы загрузить или перетащите фотографии</p>
                    <small>Поддерживаемые форматы: JPEG, PNG, WEBP</small>
                  </div>
                </div>
              )}

              <div className={styles.buttonsWrapper}>
                <Button
                  size="large"
                  type="button"
                  variant="green"
                  onClick={handleClear}
                  disabled={previewUrls.length === 0}
                >
                  Сбросить
                </Button>
                <Button
                  size="large"
                  type="submit"
                  variant="mint"
                  disabled={previewUrls.length === 0}
                >
                  Загрузить ({previewUrls.length})
                </Button>
              </div>
            </form>
          </div>
        </Card>
      </motion.div>

      <AnimatePresence>
        {showResult && (
          <motion.div
            className={styles.resultCard}
            initial={{ opacity: 0, x: 100 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 100 }}
            transition={{ duration: 0.3 }}
          >
            {isLoading ? (
              <>
                <ResultCard previews={loader} />
              </>
            ) : (
              <ResultCard previews={result} />
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
