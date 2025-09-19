interface ProcessedImage {
  id: number;
  image: string;
  created_at: string;
}

interface UploadImagesResponse {
  message: string;
  results: ProcessedImage[];
}

interface UploadImagesRequest {
  images: File[];
}
