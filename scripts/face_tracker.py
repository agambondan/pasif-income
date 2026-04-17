import cv2
import mediapipe as mp
import sys
import numpy as np

def get_face_x(video_path, start_sec, end_sec):
    mp_face_detection = mp.solutions.face_detection
    cap = cv2.VideoCapture(video_path)

    fps = cap.get(cv2.CAP_PROP_FPS)
    cap.set(cv2.CAP_PROP_POS_FRAMES, int(float(start_sec) * fps))
    end_frame = int(float(end_sec) * fps)

    x_coords = []

    with mp_face_detection.FaceDetection(model_selection=1, min_detection_confidence=0.5) as face_detection:
        while cap.isOpened():
            current_frame = int(cap.get(cv2.CAP_PROP_POS_FRAMES))
            if current_frame > end_frame:
                break

            success, image = cap.read()
            if not success:
                break

            # Sample every 10 frames to speed up processing
            if current_frame % 10 != 0:
                continue

            image.flags.writeable = False
            image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
            results = face_detection.process(image)

            if results.detections:
                for detection in results.detections:
                    bboxC = detection.location_data.relative_bounding_box
                    h, w, c = image.shape
                    # Calculate horizontal center point of the face
                    x_center = int((bboxC.xmin + bboxC.width / 2) * w)
                    x_coords.append(x_center)

    cap.release()

    if len(x_coords) == 0:
        return 1920 // 2 # Return center if no face detected

    return int(np.mean(x_coords))

if __name__ == "__main__":
    # Usage: python face_tracker.py video.mp4 10 45
    if len(sys.argv) < 4:
        print("Usage: python face_tracker.py <video_path> <start_sec> <end_sec>")
        sys.exit(1)
    path = sys.argv[1]
    start = sys.argv[2]
    end = sys.argv[3]
    print(get_face_x(path, start, end))
