import { AppOptions, cert, getApps, initializeApp } from "firebase-admin/app";
import { getFirestore, Firestore } from "firebase-admin/firestore";
import { config } from "./config";
import { logger } from "./logger";

let firestoreDb: Firestore | undefined;

export const getFirestoreDb = (): Firestore => {
  if (firestoreDb) return firestoreDb;

  const credentialMode = config.FIREBASE_SERVICE_ACCOUNT_PATH
    ? "service_account_file"
    : "application_default";
  const credential = config.FIREBASE_SERVICE_ACCOUNT_PATH
    ? cert(config.FIREBASE_SERVICE_ACCOUNT_PATH)
    : undefined;

  const existingApp = getApps()[0];
  const appOptions: AppOptions = {
    projectId: config.FIREBASE_PROJECT_ID,
  };

  if (credential) {
    appOptions.credential = credential;
  }

  const app = existingApp ?? initializeApp(appOptions);

  firestoreDb = getFirestore(app, config.FIRESTORE_DATABASE_ID);

  logger.info(
    {
      firebaseProjectId: config.FIREBASE_PROJECT_ID,
      firestoreDatabaseId: config.FIRESTORE_DATABASE_ID,
      credentialMode,
      reusedFirebaseApp: Boolean(existingApp),
    },
    "Firestore initialized",
  );

  return firestoreDb;
};

export const setFirestoreDbForTesting = (db: Firestore | undefined): void => {
  if (Bun.env.NODE_ENV !== "test") {
    throw new Error("setFirestoreDbForTesting can only be used in tests");
  }
  firestoreDb = db;
};
