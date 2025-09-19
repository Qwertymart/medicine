import styles from "./card.module.css";
import Card from "./card";
import Carousel from "../carousel/carousel";
// import LegendAccordion from "../LegendAccordion/LegendAccordion";
import DownloadMultipleImages from "../DownloadButton/downloadbutton";

// type ItemType = {
//   title: string;
//   description: string;
//   color: string;
// };

// const legendItems: ItemType[] = [
//   {
//     title: "Гарь",
//     description: "Территория, пострадавшая от лесного пожара.",
//     color: "#FF0000",
//   },
//   {
//     title: "Пашни",
//     description: "Земли, используемые для сельскохозяйственных работ.",
//     color: "#FDC611",
//   },
//   {
//     title: "Леса",
//     description: "Натуральный лесной массив, охраняемый от вырубки.",
//     color: "#00CC00",
//   },
// ];

interface ResultProps  {
  previews: string[]
}


const ResultCard: React.FC<ResultProps> = ({ previews = [] }) => {
  return (
    <div className={styles.mainContent}>
      <Card>
        <div className={styles.carouselContainer}>
          <h2 className={styles.headText}>Ваш результат</h2>

          <div className={styles.imageWrapper}>
            <Carousel previewUrls={previews} />
          </div>

          {/* <LegendAccordion legendItems={legendItems}></LegendAccordion> */}

          <div className={styles.buttonsWrapper}>
            <DownloadMultipleImages images={previews} />
          </div>
        </div>
      </Card>
    </div>
  );
};

export default ResultCard;
