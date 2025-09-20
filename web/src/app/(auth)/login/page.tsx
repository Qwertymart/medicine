/* eslint-disable @next/next/no-img-element */
"use client";
import { FormEvent, useState } from "react";
import Input from "@/components/inputs/input";
import Button from "@/components/button/button";
import { useRouter } from "next/navigation";
import classNames from "classnames";
import styles from "./page.module.css";
import LinksBlock from "@/components/LinksBlock/LinksBlock";
import { z } from "zod";
import * as m from "motion/react-m";
import { AnimatePresence } from "motion/react";

const loginSchema = z.object({
  login: z.string().nonempty("Логин обязателен"),
  password: z.string().min(6, "Пароль должен быть минимум 6 символов"),
});

export default function LoginPage() {
  const router = useRouter();
  const [errors, setErrors] = useState<{ login?: string; password?: string }>(
    {}
  );

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setErrors({});
    const formData = new FormData(e.currentTarget);

    const loginData = {
      login: formData.get("login")?.toString() ?? "",
      password: formData.get("password")?.toString() ?? "",
    };

    const validation = loginSchema.safeParse(loginData);
    if (!validation.success) {
      const fieldErrors: { login?: string; password?: string } = {};

      if (validation.error && validation.error.issues) {
        validation.error.issues.forEach((issue) => {
          if (issue.path && issue.path.length > 0) {
            const field = issue.path[0] as "login" | "password";
            fieldErrors[field] = issue.message;
          }
        });
      }
      setErrors(fieldErrors);
      return;
    }

    const response = await fetch("/api/auth", {
      method: "POST",
      body: JSON.stringify({
        action: "login",
        username: formData.get("login"),
        password: formData.get("password"),
      }),
    });

    if (response.ok) {
      router.push("/dashboard");
    } else {
      const data = await response.json();
      setErrors({ password: data.error || "Ошибка авторизации" });
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.main}>
        <form onSubmit={handleSubmit} className={styles.form}>
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

          {/* Правая колонка с формой */}
          <div className={styles.item} style={{ padding: "2rem" }}>
            <h2 className={styles.title}>Вход</h2>
            <AnimatePresence mode="wait">
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
                  variant="people"
                  placeholder="Логин"
                  required
                  className={classNames(styles.otstupiki, styles.login)}
                  name="login"
                  style={{ marginBottom: "1vh" }}
                />
                {errors.login && (
                  <p style={{ fontSize: "1rem", color: "red" }}>
                    {errors.login}
                  </p>
                )}
                <Input
                  variant="pass"
                  className={classNames(styles.otstupiki, styles.pass)}
                  placeholder="Пароль"
                  type="password"
                  name="password"
                  required
                />{" "}
                {errors.password && (
                  <p style={{ fontSize: "1rem", color: "red" }}>
                    {errors.password}
                  </p>
                )}
                <Button
                  variant="saphire"
                  size="large"
                  type="submit"
                  style={{ marginTop: "2rem" }}
                >
                  Войти
                </Button>
              </m.div>
            </AnimatePresence>
          </div>
        </form>
      </div>{" "}
    </div>
  );
}
