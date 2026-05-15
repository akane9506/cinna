/**
 * Persona loader for Cinna.
 * Reads the persona from a markdown file to keep the code clean and the prompt easy to edit.
 */

let cachedPersona: string | null = null;

/**
 * Loads the persona from the local Markdown file using Bun's native API.
 * @returns The persona string or a default fallback.
 */
export const getPersona = async (): Promise<string> => {
  if (cachedPersona) return cachedPersona;

  const file = Bun.file(`${import.meta.dir}/persona.md`);

  if (!(await file.exists())) {
    console.warn("src/core/persona.md not found, using fallback persona.");
    cachedPersona = "You're a cute assistant Cinna。";
  } else {
    cachedPersona = await file.text();
  }

  return cachedPersona;
};
