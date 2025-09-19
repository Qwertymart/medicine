import classNames from "classnames";
import styles from "./card.module.css";

interface CardProps extends React.HTMLAttributes<HTMLElement> {
  children: React.ReactNode;
}

const Card: React.FC<CardProps> = ({ children, ...props }) => {
  const cardClass = classNames(styles.card);

  return (
    <div className={cardClass} {...props}>
      {children}
    </div>
  );
};

export default Card;
