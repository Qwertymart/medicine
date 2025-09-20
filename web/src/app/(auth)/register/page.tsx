/* eslint-disable @next/next/no-img-element */
/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";
import { useState } from "react";
// import { motion, AnimatePresence } from "framer-motion";
import * as m from "motion/react-m";
import { AnimatePresence } from "motion/react";

import Input from "@/components/inputs/input";
import Button from "@/components/button/button";
import styles from "./page.module.css";
import classNames from "classnames";
import { useRouter } from "next/navigation";
import LinksBlock from "@/components/LinksBlock/LinksBlock";
import { z } from "zod";

interface RegData {
  lastName: string;
  firstName: string;
  email: string;
  login: string;
  password: string;
  confirmPassword: string;
}

const baseRegisterSchema = z.object({
  lastName: z.string().nonempty("Фамилия обязательна"),
  firstName: z.string().nonempty("Имя обязательно"),
  email: z.string().email("Неверный формат почты"),
  login: z.string().nonempty("Логин обязателен"),
  password: z.string().min(6, "Пароль должен быть минимум 6 символов"),
  confirmPassword: z.string().nonempty("Подтверждение пароля обязательно"),
});

const registerSchema = baseRegisterSchema.refine(
  (data) => data.password === data.confirmPassword,
  {
    message: "Пароли не совпадают",
    path: ["confirmPassword"],
  }
);

export default function RegisterForm() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState<RegData>({
    lastName: "",
    firstName: "",
    email: "",
    login: "",
    password: "",
    confirmPassword: "",
  });
  const [errors, setErrors] = useState<Partial<Record<keyof RegData, string>>>(
    {}
  );

  function nextStep(e: { preventDefault: () => void }) {
    e.preventDefault();
    const step1Schema = baseRegisterSchema.pick({
      lastName: true,
      firstName: true,
      email: true,
    });
    const validation = step1Schema.safeParse({
      lastName: formData.lastName,
      firstName: formData.firstName,
      email: formData.email,
    });
    if (!validation.success) {
      const newErrors: Partial<Record<keyof RegData, string>> = {};

      if (validation.error && validation.error.issues) {
        validation.error.issues.forEach((issue) => {
          if (issue.path && issue.path.length > 0) {
            const field = issue.path[0] as keyof RegData;
            if (
              Object.keys(baseRegisterSchema.shape).includes(field as string)
            ) {
              newErrors[field] = issue.message;
            }
          }
        });
      }
      setErrors(newErrors);
      return;
    }
    setErrors({});
    setStep((prev) => prev + 1);
  }

  function prevStep(e: { preventDefault: () => void }) {
    e.preventDefault();
    setStep((prev) => prev - 1);
  }

  const handleChange = (e: { target: { name: any; value: any } }) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
  };

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();
    setErrors({});
    const validation = registerSchema.safeParse(formData);
    if (!validation.success) {
      const newErrors: Partial<Record<keyof RegData, string>> = {};

      if (validation.error && validation.error.issues) {
        validation.error.issues.forEach((issue) => {
          if (issue.path && issue.path.length > 0) {
            const field = issue.path[0] as keyof RegData;
            if (
              Object.keys(baseRegisterSchema.shape).includes(field as string)
            ) {
              newErrors[field] = issue.message;
            }
          }
        });
      }
      setErrors(newErrors);
      return;
    }

    try {
      const response = await fetch("/api/auth", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          action: "register",
          last_name: formData.lastName,
          first_name: formData.firstName,
          email: formData.email,
          username: formData.login,
          password: formData.password,
          password2: formData.confirmPassword,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Ошибка регистрации");
      }

      router.push("/login");
    } catch (err: any) {
      console.error(err.message);
      setErrors({ login: err.message });
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.main}>
        {" "}
        <form className={styles.form} onSubmit={handleSubmit} method="POST">
          {/* Левая колонка с картинкой */}
          <div
            className={styles.item}
            style={{
              padding: "2rem",
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
            }}
          >
            <LinksBlock />
            <img
              src="bird_with_circle.png"
              alt="Лого"
              style={{
                marginTop: "0px",
                width: "29.02vw",
                height: "62.44vh",
                // height: "auto",
                borderRadius: "16px",
                objectFit: "cover",
              }}
            />
          </div>
          <div className={styles.item}>
            <div style={{ display: "flex", justifyContent: "center" }}>
              <div>
                {" "}
                <h2 className={styles.title}>Регистрация</h2>
              </div>
            </div>

            <AnimatePresence mode="wait">
              {step === 1 && (
                <m.div
                  key="step1"
                  initial={{ opacity: 0, x: -50 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 50 }}
                  transition={{ duration: 0.3 }}
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    width: "80%",
                  }}
                >
                  <Input
                    placeholder="Фамилия"
                    name="lastName"
                    variant="people"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.surname
                    )}
                    value={formData.lastName}
                    onChange={handleChange}
                  />
                  {errors.lastName && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.lastName}
                    </p>
                  )}
                  <Input
                    placeholder="Имя"
                    variant="people"
                    name="firstName"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.name
                    )}
                    value={formData.firstName}
                    onChange={handleChange}
                  />{" "}
                  {errors.firstName && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.firstName}
                    </p>
                  )}
                  <Input
                    placeholder="Почта"
                    variant="mail"
                    name="email"
                    type="email"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.email
                    )}
                    value={formData.email}
                    onChange={handleChange}
                  />
                  {errors.email && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.email}
                    </p>
                  )}
                  <Button
                    variant="saphire"
                    size="large"
                    onClick={nextStep}
                    style={{ marginTop: "1rem" }}
                    type="button"
                  >
                    Продолжить →
                  </Button>
                </m.div>
              )}

              {step === 2 && (
                <m.div
                  key="step2"
                  initial={{ opacity: 0, x: -50 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 50 }}
                  transition={{ duration: 0.3 }}
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    width: "80%",
                  }}
                >
                  <Input
                    placeholder="Логин"
                    name="login"
                    variant="people"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.login
                    )}
                    value={formData.login}
                    onChange={handleChange}
                  />
                  {errors.login && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.login}
                    </p>
                  )}
                  <Input
                    placeholder="Пароль"
                    name="password"
                    type="password"
                    variant="pass"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.pass1
                    )}
                    value={formData.password}
                    onChange={handleChange}
                  />
                  {errors.password && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.password}
                    </p>
                  )}
                  <Input
                    placeholder="Повторите пароль"
                    name="confirmPassword"
                    type="password"
                    variant="pass"
                    required
                    className={classNames(
                      styles.input,
                      styles.otstupiki,
                      styles.pass2
                    )}
                    value={formData.confirmPassword}
                    onChange={handleChange}
                  />
                  {errors.confirmPassword && (
                    <p style={{ fontSize: "1rem", color: "red" }}>
                      {errors.confirmPassword}
                    </p>
                  )}
                  <div className={styles.buttonGroup}>
                    <Button
                      onClick={prevStep}
                      variant="saphire"
                      size="large"
                      type="submit"
                    >
                      ← Назад
                    </Button>
                    <Button type="submit">Зарегистрироваться</Button>
                  </div>
                </m.div>
              )}
            </AnimatePresence>
          </div>
        </form>
      </div>{" "}
    </div>
  );
}
