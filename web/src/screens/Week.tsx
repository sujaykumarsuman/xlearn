import { useParams } from "react-router-dom";
import { Placeholder } from "../components/Placeholder";

export default function Week() {
  const { n } = useParams();
  return (
    <Placeholder
      eyebrow="DSA · Week"
      title={`Week ${n ?? ""}`.trim()}
      icon="layers"
      sprint="S04"
      summary="The week thesis, concept links, and the problem list with five-touch dots and a difficulty/status filter."
    />
  );
}
