import {DataTypes, sequelize} from "../config/db.js";
import User from "./userModel.js";

const KnowledgeBaseModel = sequelize.define(
    "knowledgeBase",
    {
        id: {
            type: DataTypes.INTEGER,
            autoIncrement: true,
            primaryKey: true,
        },
        file: {
            type: DataTypes.STRING,
            allowNull: false,
        },
        title: {
            type: DataTypes.STRING(500),
            allowNull: false,
        },
        description: {
            type: DataTypes.TEXT,
            allowNull: false,
        },
        createdby: {
            type: DataTypes.INTEGER,
            allowNull: false,
            references: {
                model: User,
                key: "id",
            },
        },
        createdAt: {
            type: DataTypes.DATE,
            defaultValue: DataTypes.NOW,
        },
    },
    { timestamps: false, schema: "statgate" }
);

KnowledgeBaseModel.belongsTo(User, { foreignKey: "createdby" });

// KnowledgeBaseModel.sync({ alter: true });

export default KnowledgeBaseModel;