import CameraComponent from "@/components/Camera/CameraComponent";

export default function Camera() {
  return (
    <div>
      <CameraComponent
        onPhotoTaken={function (): void {
          console.debug("alive");
        }}
      />
    </div>
  );
}
