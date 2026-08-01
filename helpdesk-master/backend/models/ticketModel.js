import { sequelize, DataTypes } from "../config/db.js";
import User from "./userModel.js";

const TicketModel = sequelize.define(
  "tickets",
  {
    id: {
      type: DataTypes.INTEGER,
      autoIncrement: true,
      primaryKey: true,
    },
    reportedby: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    level: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    facility: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    system: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    emrtype: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    category: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    status: {
      type: DataTypes.STRING,
      defaultValue: "open",
      allowNull: false,
    },
    phoneno: {
      type: DataTypes.STRING,
      allowNull: false,
    },
    description: {
      type: DataTypes.STRING(1000),
      allowNull: false,
    },
    update: {
      type: DataTypes.STRING(1000),
      allowNull: true,
    },
    module: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    dhis2module: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    dhis2instance: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    priority: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    dueDate: {
      type: DataTypes.DATE,
      allowNull: true,
    },
    resolvedDate: {
      type: DataTypes.DATE,
      allowNull: true,
    },
    imageurl: {
      type: DataTypes.STRING,
      allowNull: true,
    },
    agentId: {
      type: DataTypes.INTEGER,
      allowNull: false,
      references: {
        model: User,
        key: "id",
      },
    },
    assignedTo: {
      type: DataTypes.INTEGER,
      allowNull: true,
      references: {
        model: User,
        key: "id",
      },
    },
  },
  { timestamps: true, schema: "statgate" }
);

TicketModel.belongsTo(User, { foreignKey: "agentId" });
TicketModel.sync({ alter: true });

export default TicketModel;
