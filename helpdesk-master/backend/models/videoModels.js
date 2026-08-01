import { DataTypes } from "sequelize";
import { sequelize } from "../config/db.js";

const Video = sequelize.define(
  "Video",
  {
    title: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    video: {
      type: DataTypes.STRING,
      allowNull: true,
    },
  },
  {
    timestamps: true,
    schema: "statgate",
  }
);

export default Video;
