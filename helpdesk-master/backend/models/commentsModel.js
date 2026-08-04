import { sequelize, DataTypes } from "../config/db.js";
import Ticket from "./ticketModel.js";

const CommentModel = sequelize.define(
  "comments",
  {
    id: {
      type: DataTypes.INTEGER,
      autoIncrement: true,
      primaryKey: true,
    },
    commentedby: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    email: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    comment: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    ticketId: {
      type: DataTypes.INTEGER,
      allowNull: false,
      references: {
        model: Ticket,
        key: "id",
      },
    },
  },
  { timestamps: true, schema: "statgate" }
);

CommentModel.belongsTo(Ticket, { foreignKey: "ticketId" });

export default CommentModel;
