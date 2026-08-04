import { Sequelize, DataTypes, QueryTypes } from "sequelize";

import dotenv from "dotenv";

dotenv.config({ path: '.env' });

const credentials = {
    dialect: "postgres",
    host: process.env.POSTGRES_HOST,
    port: process.env.POSTGRES_PORT,
    database: process.env.POSTGRES_DB,
    username: process.env.POSTGRES_USER,
    password: process.env.POSTGRES_PASSWORD,
    pool: {
        max: 5,
        min: 0,
        acquire: 30000,
        idle: 10000
    },
    logging: false
};
const sequelize = new Sequelize(credentials);

const connectDB = async () => {
    try {
        await sequelize.authenticate();
        console.log("✅ Database Connection has been established successfully.");

        await sequelize.query('CREATE SCHEMA IF NOT EXISTS statgate AUTHORIZATION statgate;');
        console.log("✅ Ensured statgate schema exists.");
    } catch (error) {
        console.error("Unable to connect to the database:", error.message);
        throw error;
    }
}

export { connectDB, sequelize, Sequelize, DataTypes, QueryTypes };
