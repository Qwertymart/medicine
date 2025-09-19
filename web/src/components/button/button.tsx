import React from "react";
import classNames from "classnames";
import styles from "./button.module.css";

/**
 * вариации кнопочек по дизайну:
 * дефолтная - по умолчанию
 * saphire - рега, кнопки подробнее и тд
 * green - для подтверждения вводов
 * mint - назад
 * white - выход
 */
type ButtonVariant = "default" | "mint" | "green" | "saphire" | "white";
/**
 * Размеры
 * TODO: убрать какие-то лишние
 */
type ButtonSize = "small" | "medium" | "large";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  children: React.ReactNode;
}

const Button: React.FC<ButtonProps> = ({
  variant = "default",
  size = "medium",
  children,
  ...props
}) => {
  const buttonClass = classNames(styles.button, styles[variant], styles[size]);

  return (
    <button className={buttonClass} {...props}>
      {variant === "white" && (
        <>
          <svg
            width="37"
            height="30"
            viewBox="0 0 37 30"
            fill="currentColor"
            xmlns="http://www.w3.org/2000/svg"
            style={{marginRight: "5px"}}
          >
            <path d="M12.8696 20.7924V28.1522C12.8696 29.037 13.513 29.7609 14.3978 29.7609H35.3913C36.2761 29.7609 37 29.037 37 28.1522V1.6087C37 0.723913 36.2761 0 35.3913 0H14.3978C13.513 0 12.8696 0.723913 12.8696 1.6087V8.96848C12.8696 9.85326 13.5935 10.5772 14.4783 10.5772C15.363 10.5772 16.087 9.85326 16.087 8.96848V3.21739H33.7826V26.5435H16.087V20.7924C16.087 19.9076 15.363 19.1837 14.4783 19.1837C13.5935 19.1837 12.8696 19.9076 12.8696 20.7924ZM0.482609 13.6739L6.75652 7.31957C7.4 6.67609 8.40544 6.67609 9.04891 7.31957C9.69239 7.96304 9.69239 8.96848 9.04891 9.61196L5.46956 13.2315L24.975 13.2717C25.8598 13.2717 26.5837 13.9957 26.5837 14.8804C26.5837 15.7652 25.8598 16.4891 24.975 16.4891L5.46956 16.4489L9.04891 20.0685C9.69239 20.712 9.65217 21.7174 9.04891 22.3609C8.72717 22.6826 8.325 22.8435 7.92282 22.8435C7.52065 22.8435 7.07826 22.6826 6.79674 22.3609L0.482609 16.0065C-0.16087 15.3228 -0.16087 14.3174 0.482609 13.6739Z" />
          </svg>
        </>
      )}
      {children}
    </button>
  );
};

export default Button;
