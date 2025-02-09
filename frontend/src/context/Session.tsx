import React, { useState } from "react";
import { Alert } from "react-native";
import { API } from "../api/api";
import { Type } from "../types/user";
import { HTTPStatus } from "../utils/httpStatus";
import { Session } from "../utils/session";

// Themes: https://www.youtube.com/watch?v=km1qm1Zz2lY

export type UserSession = {
	id: string,
	username: string,
	email: string,
	verified_email: boolean,
	profile_image_url?: string,
	type: Type,
}

export type loginFunc = ({ username, password }: { username?: string, password?: string }) => void

export type Login = {
	username: string,
	password: string,
	device_token: string,
}

type SessionProps = {
	user: UserSession | null;
	login: loginFunc;
	logout: () => void;
	// Maybe languages should be persisted (an in other context called Settings)
	language: "en" | "es" | "ch";
}

export const SessionContext = React.createContext<SessionProps>({
	user: null,
	login: () => { },
	logout: () => { },
	language: "en",
});

interface Props {
	children: React.ReactNode;
}

export const AuthProvider = (props: Props) => {
	const [user, setUser] = useState<UserSession | null>(null);

	return (
		<SessionContext.Provider
			value={{
				user,
				login: async ({ username, password }) => {
					if (user !== null) {
						return;
					}
					if (!username) {
						Alert.alert("Error", "Invalid username");
						return;
					} else if (!password) {
						Alert.alert("Error", "Invalid password");
						return;
					}

					try {
						const deviceToken = ""; //await Notifications.getToken();
						const login: Login = {
							username: username.trim(),
							password: password.trim(),
							device_token: deviceToken ? deviceToken : "",
						};

						const res = await API.Auth.Login(login);
						switch (res.status) {
							case HTTPStatus.NO_CONTENT:
								const storedUser = await Session.get();
								if (storedUser) {
									setUser(storedUser);
								}
								break;
							case HTTPStatus.OK:
								const u = await res.json() as UserSession;
								setUser(u);
								await Session.save(u, username, password);
								break;
						}
					} catch (err) {
						console.log("login error:", err);
					}
				},
				logout: async () => {
					try {
						await API.Auth.Logout();
						setUser(null);
						Session.remove();
					} catch (err) {
						console.log("logout error:", err);
					}
				},
				language: "en",
			}}
		>
			{props.children}
		</SessionContext.Provider>
	);
};

