import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import styles from "./u_banner.module.css";
import { useRouter } from "next/navigation";

interface UnauthorizedToastProps {
  message: string;
  visible: boolean;
  onClose?: () => void;
}

const UnauthorizedToast: React.FC<UnauthorizedToastProps> = ({
  message,
  visible,
  onClose,
}) => {
  const router = useRouter();

  const handleLogin = () => {
    router.push("/login");
  };

    const handleMain = () => {
      router.push("/");
    };

  useEffect(() => {
    if (visible) {
      const timer = setTimeout(() => {
        if (onClose) onClose();
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [visible, onClose]);

  return (
    <AnimatePresence>
      {visible && (
        <>
          <motion.div
            className={styles.overlay}
            initial={{ opacity: 0 }}
            animate={{ opacity: 0.5 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
          />
          <motion.div
            className={styles.modal}
            initial={{ opacity: 0, x: "100%" }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: "100%" }}
            transition={{ duration: 0.5, ease: "easeInOut" }}
          >
            <h3>Доступ ограничен</h3>
            <p>{message}</p>
            <div className={styles.buttonGroup}>
              <button className={styles.loginButton} onClick={handleLogin}>
                Войти
              </button>
              <button className={styles.mainButton} onClick={handleMain}>
                На главную
              </button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};

export default UnauthorizedToast;
