/**
 * 3 разновидности
 * иконка -- пренадлежность:
 * челикс - фио, логин
 * письмо - почта
 * ключ - пароль
 */
"use client";

import classNames from "classnames";
import styles from "./input.module.css";

// Определяем возможные варианты инпута
type InputVariant = "default" | "people" | "mail" | "pass";

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  variant?: InputVariant;
  className: string,
  placeholder: string,
}

const Input: React.FC<InputProps> = ({
  variant = "default",
  className,
  placeholder,
  ...props
}) => {
  const mailPlaceholder = variant === "mail" ? "Введите email" : placeholder;

  return (
    <div
      className={classNames(styles.inputWrapper, styles[variant], className)}
    >
      <span
        className={classNames(styles.icon, styles[`icon-${variant}`])}
      ></span>
      <input className={styles.input} {...props} placeholder={mailPlaceholder} />
    </div>
  );
};

export default Input;
