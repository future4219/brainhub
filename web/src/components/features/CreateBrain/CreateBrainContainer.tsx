import {
  type ChangeEvent,
  type FormEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import { CreateBrainPresenter } from "@/components/features/CreateBrain/CreateBrainPresenter";
import {
  createBrainFailure,
  type CreateBrainFailure,
  sourceIDCandidate,
} from "@/components/features/CreateBrain/createBrainForm";
import { brainAddressPrefix, brainUrl } from "@/config/url";
import type { CreateBrainInput } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { APIError } from "@/lib/api";
import { createBrain, getPublicConfig } from "@/lib/brainApi";

export function CreateBrainContainer() {
  const navigate = useNavigate();
  const shell = useViewer();
  const submittingRef = useRef(false);
  const sourceIDEdited = useRef(false);
  const [name, setName] = useState("");
  const [sourceID, setSourceID] = useState("");
  const [addressPrefix, setAddressPrefix] = useState("");
  const [configError, setConfigError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<CreateBrainFailure | null>(null);

  useEffect(() => {
    document.title = "新しい脳を作る — brainhub";
    void getPublicConfig()
      .then((config) => setAddressPrefix(brainAddressPrefix(config.web_url)))
      .catch(() => setConfigError(true));
  }, []);

  function changeName(event: ChangeEvent<HTMLInputElement>) {
    const value = event.currentTarget.value;
    setName(value);
    if (!sourceIDEdited.current) setSourceID(sourceIDCandidate(value));
  }

  function changeSourceID(event: ChangeEvent<HTMLInputElement>) {
    sourceIDEdited.current = true;
    setSourceID(event.currentTarget.value);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current) return;
    submittingRef.current = true;
    setSubmitting(true);
    setError(null);
    const data = new FormData(event.currentTarget);
    const input: CreateBrainInput = {
      source_id: String(data.get("source_id")),
      name: String(data.get("name")),
      description: String(data.get("description")),
      visibility: String(
        data.get("visibility"),
      ) as CreateBrainInput["visibility"],
    };
    try {
      const brain = await createBrain(input);
      navigate(brainUrl(brain.source_id), { replace: true });
    } catch (cause) {
      setError(
        createBrainFailure(cause instanceof APIError ? cause : undefined),
      );
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  return (
    <CreateBrainPresenter
      {...shell}
      name={name}
      sourceID={sourceID}
      addressPrefix={addressPrefix}
      configError={configError}
      submitting={submitting}
      error={error}
      onNameChange={changeName}
      onSourceIDChange={changeSourceID}
      onSubmit={submit}
    />
  );
}
