/** Maps a MAA client type to the PC client window title. */
export const gameWindowTitle = (clientType: string) => {
  switch (clientType) {
    case "YoStarEN":
      return "Arknights";
    case "YoStarJP":
      return "アークナイツ";
    case "YoStarKR":
      return "명일방주";
    default:
      // Official, Bilibili and txwy all use the CN client window title.
      return "明日方舟";
  }
};
